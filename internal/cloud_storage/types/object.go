package types

import (
	"context"

	"github.com/tscrond/dropper/internal/filedata"
)

type ObjectStorage interface {
	SendFileToBucket(ctx context.Context, data *filedata.FileData) error
	BucketExists(ctx context.Context, fullBucketName string) (bool, error)
	CreateBucketIfNotExists(ctx context.Context, userId string) error
	GetUserBucketData(ctx context.Context, id string) (any, error)
	GetUserBucketName(ctx context.Context) (string, error)
	GenerateSignedURL(ctx context.Context, bucket, object string) (string, error)
	Close() error
}
