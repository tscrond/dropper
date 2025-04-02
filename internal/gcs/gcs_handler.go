package gcs

import (
	"context"
	"io"
	"log"

	"cloud.google.com/go/storage"
	"github.com/tscrond/dropper/internal/filedata"
	"google.golang.org/api/option"
)

type GCSBucketHandler struct {
	ServiceAccountKeyPath string
	BucketName            string
}

func NewGCSBucketHandler(svcaccountPath, bucketName string) *GCSBucketHandler {

	return &GCSBucketHandler{
		ServiceAccountKeyPath: svcaccountPath,
		BucketName:            bucketName,
	}
}

// Handler that processes a single file per request
func (b *GCSBucketHandler) SendFileToBucket(ctx context.Context, data *filedata.FileData) error {
	if data == nil {
		log.Println("Data for bucket operation is empty")
		return nil
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(b.ServiceAccountKeyPath))
	if err != nil {
		log.Println("Error initializing client:", err)
		return err
	}
	defer client.Close()

	fileName := data.RequestHeaders.Filename

	writer := client.Bucket(b.BucketName).Object(fileName).NewWriter(ctx)
	if _, err := io.Copy(writer, data.MultipartFile); err != nil {
		log.Println("Error uploading file")
		return err
	}

	if err := writer.Close(); err != nil {
		log.Println("Error closing writer:", err)
		return err
	}

	log.Printf("File %s uploaded successfully", fileName)
	return nil
}
