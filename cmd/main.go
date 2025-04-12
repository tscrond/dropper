package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tscrond/dropper/internal/api"
	"github.com/tscrond/dropper/internal/cloud_storage/factory"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {

	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	frontendEndpoint := os.Getenv("FRONTEND_ENDPOINT")
	backendEndpoint := os.Getenv("BACKEND_ENDPOINT")
	// for now - static GCS provider - later implementing min.io for self hosting
	storageProvider := "gcs"

	log.Printf("%s", fmt.Sprintf("%s/auth/callback", backendEndpoint))

	bucketHandler, err := factory.NewStorageProvider(storageProvider)
	if err != nil {
		log.Fatalln(err)
	}
	defer bucketHandler.Close()

	s := api.NewAPIServer(":3000", frontendEndpoint, bucketHandler, &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/callback", backendEndpoint),
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	})

	s.Start()
}
