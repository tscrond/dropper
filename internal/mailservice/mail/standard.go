package mailservice

import (
	mailtypes "github.com/tscrond/dropper/internal/mailservice/types"
	"github.com/tscrond/dropper/internal/repo"
)

type StandardEmailService struct {
	repository *repo.Repository
}

func NewStandardMailService(cfg mailtypes.StandardSenderConfig, r *repo.Repository) (*StandardEmailService, error) {
	return &StandardEmailService{
		repository: r,
	}, nil
}

func (s *StandardEmailService) Send(config mailtypes.MessageConfig) (any, error) {
	return nil, nil
}
