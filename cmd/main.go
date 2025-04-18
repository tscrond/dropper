package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/tscrond/dropper/internal/api"
	"github.com/tscrond/dropper/internal/cloud_storage/factory"
	"github.com/tscrond/dropper/internal/cloud_storage/types"
	"github.com/tscrond/dropper/internal/repo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	frontendEndpoint := os.Getenv("FRONTEND_ENDPOINT")
	backendEndpoint := os.Getenv("BACKEND_ENDPOINT")
	connStr := os.Getenv("DB_CONNECTION_STRING")

	// for now - static GCS provider - later implementing min.io for self hosting
	storageProvider := "gcs"

	bucketHandler, err := InitObjectStorage(backendEndpoint, storageProvider)
	if err != nil {
		log.Fatalln(err)
	}
	defer bucketHandler.Close()

	repository, err := InitRepository(connStr)
	if err != nil {
		log.Fatalln(err)
	}
	defer repository.Close()

	s := api.NewAPIServer(":3000", frontendEndpoint, bucketHandler, repository, &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/callback", backendEndpoint),
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	})

	s.Start()
}

func InitObjectStorage(backendEndpoint, storageProvider string) (types.ObjectStorage, error) {

	log.Printf("%s", fmt.Sprintf("%s/auth/callback", backendEndpoint))

	return factory.NewStorageProvider(storageProvider)
}

func InitRepository(connString string) (*repo.Repository, error) {
	if connString == "" {
		panic("no conn string provided")
	}

	fmt.Println("conn str:", connString)

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	return repo.NewRepository(db)
}
