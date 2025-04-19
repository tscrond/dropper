package factory

import (
	"errors"
	"log"
	"os"

	"github.com/tscrond/dropper/internal/cloud_storage/gcs"
	"github.com/tscrond/dropper/internal/cloud_storage/types"
	"github.com/tscrond/dropper/internal/repo"
)

func NewStorageProvider(provider string, repository *repo.Repository) (types.ObjectStorage, error) {
	switch provider {
	case "gcs":
		bucketName := os.Getenv("GCS_BUCKET_NAME")
		svcaccountPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		googleProjectID := os.Getenv("GOOGLE_PROJECT_ID")

		log.Println("bucket base name:", bucketName)
		log.Println("google credential file:", svcaccountPath)

		return gcs.NewGCSBucketHandler(svcaccountPath, bucketName, googleProjectID, repository)
	case "minio":
		return nil, errors.New("not implemented")
	default:
		panic("unknown storage type")
	}
}
