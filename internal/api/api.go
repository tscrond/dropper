package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"github.com/tscrond/dropper/internal/cloud_storage/types"
	"github.com/tscrond/dropper/internal/repo"
	"golang.org/x/oauth2"
)

type APIServer struct {
	listenPort       string
	bucketHandler    types.ObjectStorage
	repository       *repo.Repository
	OAuthConfig      *oauth2.Config
	frontendEndpoint string
}

func NewAPIServer(lp string, fe string, bh types.ObjectStorage, repository *repo.Repository, oauth2conf *oauth2.Config) *APIServer {
	return &APIServer{
		listenPort:       lp,
		frontendEndpoint: fe,
		bucketHandler:    bh,
		repository:       repository,
		OAuthConfig:      oauth2conf,
	}
}

func (s *APIServer) Start() {

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{s.frontendEndpoint},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	r.Use(c.Handler)

	// functionality
	r.Handle("/upload", s.authMiddleware(http.HandlerFunc(s.uploadHandler)))

	// auth
	r.Handle("/auth/callback", http.HandlerFunc(s.authCallback))
	r.Handle("/auth/oauth", http.HandlerFunc(s.oauthHandler))
	r.Handle("/auth/is_valid", http.HandlerFunc(s.isValid))
	r.Handle("/auth/logout", http.HandlerFunc(s.logout))

	// data ops
	r.Handle("/user_data", s.authMiddleware(http.HandlerFunc(s.getUserData)))
	r.Handle("/bucket", s.authMiddleware(http.HandlerFunc(s.getUserBucketData)))
	r.Handle("/share_with", s.authMiddleware(http.HandlerFunc(s.shareWith)))

	log.Printf("Listening on %s\n", s.listenPort)
	http.ListenAndServe("0.0.0.0"+s.listenPort, r)
}
