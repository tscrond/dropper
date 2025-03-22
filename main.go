package main

import (
	"log"
	"os"
)

func main() {
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	svcaccountPath := os.Getenv("GCS_SVCACCOUNT_PATH")

	log.Println(bucketName, svcaccountPath)
	
	bucketHandler := NewGCSBucketHandler(svcaccountPath, bucketName)

	s := NewAPIServer(":3000", bucketHandler)
	s.Start()
}
