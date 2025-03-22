package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

type APIServer struct {
	listenPort    string
	bucketHandler *GCSBucketHandler
}

func NewAPIServer(lp string, bh *GCSBucketHandler) *APIServer {
	return &APIServer{
		listenPort:    lp,
		bucketHandler: bh,
	}
}

func (s *APIServer) Start() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "multipart/form-data"},
		AllowCredentials: true,
	})

	r.Use(c.Handler)
	r.Handle("/upload", http.HandlerFunc(s.uploadHandler))

	log.Printf("Listening on %s\n", s.listenPort)
	http.ListenAndServe(s.listenPort, r)
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
