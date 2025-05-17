package factory

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	mailservice "github.com/tscrond/dropper/internal/mailservice/mail"
	"github.com/tscrond/dropper/internal/mailservice/types"
	"github.com/tscrond/dropper/internal/repo"
)

func NewEmailService(provider string, repository *repo.Repository) (types.EmailSender, error) {
	switch provider {
	case "ses":
		awsRegion := os.Getenv("AWS_REGION")
		accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
		secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion), config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyId, secretAccessKey, ""),
		))
		if err != nil {
			return nil, err
		}

		return mailservice.NewSESEmailService(cfg, repository)
	case "standard":
		// config here
		cfg := types.StandardSenderConfig{}
		return mailservice.NewStandardMailService(cfg, repository)
	case "other":
		return nil, errors.New("not implemented")

	default:
		panic("unknown storage type")
	}
}
