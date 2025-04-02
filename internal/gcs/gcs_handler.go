package gcs

import (
	"context"
	"fmt"
	"io"
	"log"

	"cloud.google.com/go/storage"
	"github.com/tscrond/dropper/internal/filedata"
	"github.com/tscrond/dropper/internal/userdata"
	"google.golang.org/api/option"
)

// TODO create named buckets <bucket_name>-<user_id> + restrict access by ID and token verification
type GCSBucketHandler struct {
	ServiceAccountKeyPath string
	BucketName            string
	GoogleProjectID       string
}

func NewGCSBucketHandler(svcaccountPath, bucketName, projId string) *GCSBucketHandler {

	return &GCSBucketHandler{
		ServiceAccountKeyPath: svcaccountPath,
		BucketName:            bucketName,
		GoogleProjectID:       projId,
	}
}

// Handler that processes a single file per request
func (b *GCSBucketHandler) SendFileToBucket(ctx context.Context, data *filedata.FileData) error {
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)

	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		return nil
	}

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

	userBucketName := fmt.Sprintf("%s-%s", b.BucketName, authUserData.Id)

	if err := b.CreateBucketIfNotExists(ctx, userBucketName, authUserData.Id); err != nil {
		log.Println(err)
		return err
	}

	writer := client.Bucket(userBucketName).Object(fileName).NewWriter(ctx)
	if _, err := io.Copy(writer, data.MultipartFile); err != nil {
		log.Println("error uploading file: ", err)
		return err
	}

	if err := writer.Close(); err != nil {
		log.Println("error closing writer:", err)
		return err
	}

	log.Printf("file %s uploaded successfully", fileName)
	return nil
}

// func (b *GCSBucketHandler) AttachPoliciesToBucket(ctx context.Context, client *storage.Client, bucketName, userEmail string) error {
// 	bucket := client.Bucket(bucketName)
// 	policy, err := bucket.IAM().Policy(ctx)

// 	if err != nil {
// 		return fmt.Errorf("error getting bucket IAM policy: %v", err)
// 	}

// 	role := iam.RoleName("roles/storage.objectAdmin")
// 	member := fmt.Sprintf("user:%s", userEmail)

// 	policy.Add(member, role)
// 	if err := bucket.IAM().SetPolicy(ctx, policy); err != nil {
// 		return fmt.Errorf("failed to update IAM policy: %v", err)
// 	}

// 	fmt.Printf("Granted %s access to bucket %s\n", userEmail, bucketName)
// 	return nil
// }

func (b *GCSBucketHandler) bucketExists(ctx context.Context, client *storage.Client, fullBucketName string) (bool, error) {
	_, err := client.Bucket(fullBucketName).Attrs(ctx)
	if err == storage.ErrBucketNotExist {
		return false, nil
	}
	return err == nil, err
}

func (b *GCSBucketHandler) CreateBucketIfNotExists(ctx context.Context, bucketName, userId string) error {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %v", err)
	}
	defer storageClient.Close()

	exists, err := b.bucketExists(ctx, storageClient, bucketName)
	if !exists {
		if err := createBucket(ctx, storageClient, bucketName, b.GoogleProjectID); err != nil {
			log.Println("error creating storage bucket: ", err)
			return err
		}
		return nil
	}
	if err != nil {
		log.Println("error checking for bucket: ", err)
		return err
	}

	return nil
}

func createBucket(ctx context.Context, client *storage.Client, fullBucketName, projectID string) error {
	bucket := client.Bucket(fullBucketName)
	err := bucket.Create(ctx, projectID, &storage.BucketAttrs{
		Location: "europe-west1",
	})
	if err != nil {
		return err
	}
	log.Printf("bucket %s created successfully", fullBucketName)
	return nil
}
