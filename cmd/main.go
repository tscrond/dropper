package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/microcosm-cc/bluemonday"

	"github.com/tscrond/dropper/internal/api"
	"github.com/tscrond/dropper/internal/cloud_storage/factory"
	"github.com/tscrond/dropper/internal/cloud_storage/types"
	"github.com/tscrond/dropper/internal/config"
	"github.com/tscrond/dropper/internal/repo"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	listenPort := os.Getenv("DROPPER_LISTEN_PORT")
	if listenPort == "" {
		listenPort = "3000"
	}
	clientId := os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	frontendEndpoint := os.Getenv("FRONTEND_ENDPOINT")
	backendEndpoint := os.Getenv("BACKEND_ENDPOINT")

	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	//postgres://<user>:<pass>@<dbhost>:5432/<dbname>?sslmode=disable
	connStr := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbName)

	// log.Printf("db connection string: %s", connStr)
	log.Printf("backend endpoint: %s\n frontend endpoint: %s", backendEndpoint, frontendEndpoint)

	repository, err := InitRepository(connStr)
	if err != nil {
		log.Fatalln(err)
	}
	defer repository.Close()

	// for now - static GCS provider - later implementing min.io for self hosting
	storageProvider := "gcs"

	bucketHandler, err := InitObjectStorage(backendEndpoint, storageProvider, repository)
	if err != nil {
		log.Fatalln(err)
	}
	defer bucketHandler.Close()

	htmlSanitizationPolicy := bluemonday.UGCPolicy()

	backendConfig := config.BackendConfig{
		ListenPort:             fmt.Sprintf(":%s", listenPort),
		BackendEndpoint:        backendEndpoint,
		FrontendEndpoint:       frontendEndpoint,
		HTMLSanitizationPolicy: htmlSanitizationPolicy,
	}

	s := api.NewAPIServer(backendConfig, bucketHandler, repository, &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("%s/auth/callback", backendEndpoint),
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	})

	s.Start()
}

func InitObjectStorage(backendEndpoint, storageProvider string, repository *repo.Repository) (types.ObjectStorage, error) {

	log.Printf("%s", fmt.Sprintf("%s/auth/callback", backendEndpoint))

	return factory.NewStorageProvider(storageProvider, repository)
}

func InitRepository(connString string) (*repo.Repository, error) {
	if connString == "" {
		panic("no conn string provided")
	}

	// log.Println("conn str:", connString)

	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	return repo.NewRepository(db)
}
