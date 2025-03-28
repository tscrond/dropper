package main

import (
	"log"
	"os"

	"github.com/markbates/goth/providers/google"
	"golang.org/x/oauth2"
)

func main() {
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	svcaccountPath := os.Getenv("GCS_SVCACCOUNT_PATH")
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")

	log.Println(bucketName, svcaccountPath)

	bucketHandler := NewGCSBucketHandler(svcaccountPath, bucketName)

	s := NewAPIServer(":3000", bucketHandler, &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  "http://localhost:3000/auth/callback",
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	})

	s.Start()
}
