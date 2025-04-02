package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tscrond/dropper/internal/api"
	"github.com/tscrond/dropper/internal/gcs"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	svcaccountPath := os.Getenv("GCS_SVCACCOUNT_PATH")
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	frontendEndpoint := os.Getenv("FRONTEND_ENDPOINT")
	backendEndpoint := os.Getenv("BACKEND_ENDPOINT")

	log.Println(bucketName, svcaccountPath)
	log.Printf("%s", fmt.Sprintf("%s/auth/callback", backendEndpoint))

	bucketHandler := gcs.NewGCSBucketHandler(svcaccountPath, bucketName)

	s := api.NewAPIServer(":3000", frontendEndpoint, bucketHandler, &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/callback", backendEndpoint),
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	})

	s.Start()
}
