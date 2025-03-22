package main

import (
	"context"
	"io"
	"log"
	"mime/multipart"

	"cloud.google.com/go/storage"
)

type GCSBucketHandler struct {
	ServiceAccountKeyPath string
	BucketName            string
}

type FileData struct {
	multipartFile  multipart.File
	requestHeaders *multipart.FileHeader
}

func NewFileData(multipartFile multipart.File, requestHeaders *multipart.FileHeader) *FileData {

	// if requestHeaders != nil {
	// 	log.Println("Request headers are empty")
	// 	return nil
	// }

	return &FileData{
		multipartFile:  multipartFile,
		requestHeaders: requestHeaders,
	}
}

func NewGCSBucketHandler(svcaccountPath, bucketName string) *GCSBucketHandler {

	return &GCSBucketHandler{
		ServiceAccountKeyPath: svcaccountPath,
		BucketName:            bucketName,
	}
}

// Handler that processes a single file per request
func (b *GCSBucketHandler) SendFileToBucket(ctx context.Context, data *FileData) error {
	if data == nil {
		log.Println("Data for bucket operation is empty")
		return nil
	}

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Println("Error initializing client:", err)
		return err
	}
	defer client.Close()

	fileName := data.requestHeaders.Filename

	writer := client.Bucket(b.BucketName).Object(fileName).NewWriter(ctx)
	if _, err := io.Copy(writer, data.multipartFile); err != nil {
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
