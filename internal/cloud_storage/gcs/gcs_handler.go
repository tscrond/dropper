package gcs

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"

	"github.com/tscrond/dropper/internal/cloud_storage/types"
	"github.com/tscrond/dropper/internal/filedata"
	"github.com/tscrond/dropper/internal/userdata"
	"github.com/tscrond/dropper/pkg"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// TODO create named buckets <bucket_name>-<user_id> + restrict access by ID and token verification
type GCSBucketHandler struct {
	Client                *storage.Client
	ServiceAccountKeyPath string
	BaseBucketName        string
	GoogleProjectID       string
}

func NewGCSBucketHandler(svcaccountPath, bucketName, projId string) (types.ObjectStorage, error) {

	var err error
	for i := 0; i < 5; i++ {
		_, err = os.Stat(svcaccountPath)
		if err == nil {
			break
		}
		log.Printf("Retrying to find credentials file (%s): %v", svcaccountPath, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Printf("Failed to find credentials file after retries: %v\n", err)
		return nil, err
	}

	client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(svcaccountPath))
	if err != nil {
		log.Println("Error initializing client:", err)
		return nil, err
	}

	return &GCSBucketHandler{
		Client:                client,
		ServiceAccountKeyPath: svcaccountPath,
		BaseBucketName:        bucketName,
		GoogleProjectID:       projId,
	}, nil
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

	fileName := data.RequestHeaders.Filename

	if err := b.CreateBucketIfNotExists(ctx, authUserData.Id); err != nil {
		log.Println(err)
		return err
	}

	userBucketName := pkg.GetUserBucketName(b.BaseBucketName, authUserData.Id)

	writer := b.Client.Bucket(userBucketName).Object(fileName).NewWriter(ctx)
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

func (b *GCSBucketHandler) BucketExists(ctx context.Context, fullBucketName string) (bool, error) {
	_, err := b.Client.Bucket(fullBucketName).Attrs(ctx)
	if err == storage.ErrBucketNotExist {
		return false, nil
	}
	return err == nil, err
}

func (b *GCSBucketHandler) CreateBucketIfNotExists(ctx context.Context, userId string) error {

	bucketName := pkg.GetUserBucketName(b.BaseBucketName, userId)

	exists, err := b.BucketExists(ctx, bucketName)
	if !exists {
		if err := b.CreateBucket(ctx, bucketName, b.GoogleProjectID); err != nil {
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

func (b *GCSBucketHandler) GetUserBucketData(ctx context.Context, id string) (any, error) {

	bucketName := pkg.GetUserBucketName(b.BaseBucketName, id)
	it := b.Client.Bucket(bucketName).Objects(ctx, nil)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err)
		}
		log.Printf("%+v\n", objAttrs)
	}
	return BucketData{}, nil
}

func (b *GCSBucketHandler) CreateBucket(ctx context.Context, fullBucketName, projectID string) error {
	bucket := b.Client.Bucket(fullBucketName)
	// userData, _ := ctx.Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)

	err := bucket.Create(ctx, projectID, &storage.BucketAttrs{
		Location: "europe-west1",
	})
	if err != nil {
		return err
	}

	log.Printf("bucket %s created successfully", fullBucketName)
	return nil
}

func (b *GCSBucketHandler) Close() error {
	if b.Client != nil {
		return b.Client.Close()
	}
	return nil
}
