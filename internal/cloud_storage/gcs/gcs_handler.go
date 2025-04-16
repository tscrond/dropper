package gcs

import (
	"context"
	"errors"
	"fmt"
	"html"
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

func (b *GCSBucketHandler) getBucketAttrs(ctx context.Context, bucketName string) (*BucketData, error) {
	bucketDataAttrs, err := b.Client.Bucket(bucketName).Attrs(ctx)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, fmt.Errorf("Bucket(%q).Attrs: %w", bucketName, err)
	}
	fmt.Printf("BucketName: %v\n", bucketDataAttrs.Name)
	fmt.Printf("StorageClass: %v\n", bucketDataAttrs.StorageClass)
	fmt.Printf("TimeCreated: %v\n", bucketDataAttrs.Created)
	if bucketDataAttrs.Labels != nil {
		fmt.Printf("\n\n\nLabels:")
		for key, value := range bucketDataAttrs.Labels {
			fmt.Printf("\t%v = %v\n", key, value)
		}
	}

	return &BucketData{
		BucketName:   bucketDataAttrs.Name,
		StorageClass: bucketDataAttrs.StorageClass,
		TimeCreated:  bucketDataAttrs.Created,
		Labels:       bucketDataAttrs.Labels,
	}, nil
}

func (b *GCSBucketHandler) getObjectsAttrs(ctx context.Context, bucketName string) ([]ObjectMedatata, error) {
	var objects []ObjectMedatata
	it := b.Client.Bucket(bucketName).Objects(ctx, nil)
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Println(err)
			continue
		}
		// log.Printf("%+v\n", objAttrs)

		objects = append(objects, ObjectMedatata{
			Name:        objAttrs.Name,
			ContentType: objAttrs.ContentType,
			Created:     objAttrs.Created,
			Deleted:     objAttrs.Deleted,
			Updated:     objAttrs.Updated,
			MD5:         objAttrs.MD5,
			Size:        objAttrs.Size,
			MediaLink:   objAttrs.MediaLink,
			Bucket:      objAttrs.Bucket,
		})
	}

	return objects, nil
}

func (b *GCSBucketHandler) getObjectsAttrsByObjName(ctx context.Context, bucketName, objName string) (*ObjectMedatata, error) {
	var selectedObj *ObjectMedatata
	objects, err := b.getObjectsAttrs(ctx, bucketName)
	if err != nil {
		log.Println("error getting objects attributes", err)
		return nil, err
	}
	for _, o := range objects {
		if o.Name == objName {
			selectedObj = &o
		}
	}
	return selectedObj, nil
}

func (b *GCSBucketHandler) GetUserBucketData(ctx context.Context, id string) (any, error) {

	bucketName := pkg.GetUserBucketName(b.BaseBucketName, id)

	bucketData, err := b.getBucketAttrs(ctx, bucketName)
	if err != nil {
		log.Println("error getting bucket metadata: ", err)
		return nil, err
	}

	objects, err := b.getObjectsAttrs(ctx, bucketName)
	if err != nil {
		log.Println("error getting objects metadata: ", err)
		return nil, err
	}

	bucketData.Objects = objects

	return bucketData, nil
}

func (b *GCSBucketHandler) GetUserBucketName(ctx context.Context) (string, error) {
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)

	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		return "", errors.New("cannot read authorized user data")
	}

	bucketName := pkg.GetUserBucketName(b.BaseBucketName, authUserData.Id)

	return bucketName, nil
}

func (b *GCSBucketHandler) CreateBucket(ctx context.Context, fullBucketName, projectID string) error {
	bucket := b.Client.Bucket(fullBucketName)
	// userData, _ := ctx.Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)

	err := bucket.Create(ctx, projectID, &storage.BucketAttrs{
		Location: "europe-west1",
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true,
		},
		PublicAccessPrevention: storage.PublicAccessPreventionEnforced,
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

func (b *GCSBucketHandler) GenerateSignedURL(ctx context.Context, bucket, object string) (string, error) {

	email, pkey, err := pkg.LoadServiceAccount(b.ServiceAccountKeyPath)
	if err != nil {
		return "", fmt.Errorf("Bucket(%q) error reading svc account: %w", bucket, err)
	}

	// attrs, _ := b.getObjectsAttrsByObjName(ctx, bucket, object)

	u, err := storage.SignedURL(bucket, object, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		Expires:        time.Now().Add(15 * time.Minute),
		GoogleAccessID: email,
		PrivateKey:     pkey,
		// Style:          storage.VirtualHostedStyle(),
	})

	if err != nil {
		return "", fmt.Errorf("Bucket(%q).SignedURL: %w", bucket, err)
	}

	u = html.UnescapeString(u)

	fmt.Println(u)

	return u, nil
}
