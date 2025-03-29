package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"golang.org/x/oauth2"
)

type APIServer struct {
	listenPort       string
	bucketHandler    *GCSBucketHandler
	OAuthConfig      *oauth2.Config
	frontendEndpoint string
}

func NewAPIServer(lp string, fe string, bh *GCSBucketHandler, oauth2conf *oauth2.Config) *APIServer {
	return &APIServer{
		listenPort:       lp,
		frontendEndpoint: fe,
		bucketHandler:    bh,
		OAuthConfig:      oauth2conf,
	}
}

func (s *APIServer) Start() {

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{s.frontendEndpoint},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	r.Use(c.Handler)

	r.Handle("/upload", s.AuthMiddleware(c.Handler(http.HandlerFunc(s.uploadHandler))))

	r.Handle("/auth/callback", c.Handler(http.HandlerFunc(s.authCallback)))

	r.Handle("/auth/oauth", c.Handler(http.HandlerFunc(s.oauthHandler)))

	r.Handle("/auth/is_valid", c.Handler(http.HandlerFunc(s.isValid)))

	// r.Handle("/logout/{provider}", c.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	gothic.Logout(w, r)
	// 	w.Header().Set("Location", "/")
	// 	w.WriteHeader(http.StatusTemporaryRedirect)
	// })))

	log.Printf("Listening on %s\n", s.listenPort)
	http.ListenAndServe("0.0.0.0"+s.listenPort, r)
}

func (s *APIServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get file from request
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to parse file from request", http.StatusBadRequest)
		log.Println(err)
		return
	}
	defer file.Close()

	// Create fileData object
	fileData := NewFileData(file, header)
	if fileData == nil {
		http.Error(w, "Invalid file data", http.StatusInternalServerError)
		log.Println("Invalid file data")
		return
	}

	if err := s.bucketHandler.SendFileToBucket(ctx, fileData); err != nil {
		http.Error(w, "Failed to send file to bucket", http.StatusInternalServerError)
		log.Fatal(err)
	}

	fmt.Fprintf(w, "Files uploaded successfully: %+v\n", fileData.requestHeaders.Filename)
}
