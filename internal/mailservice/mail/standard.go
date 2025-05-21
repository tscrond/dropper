package mailservice

import (
	"fmt"
	"net/smtp"

	mailtypes "github.com/tscrond/dropper/internal/mailservice/types"
	"github.com/tscrond/dropper/internal/repo"
)

type StandardEmailService struct {
	config     *mailtypes.StandardSenderConfig
	repository *repo.Repository
}

func NewStandardMailService(cfg *mailtypes.StandardSenderConfig, r *repo.Repository) (*StandardEmailService, error) {
	return &StandardEmailService{
		config:     cfg,
		repository: r,
	}, nil
}

func (s *StandardEmailService) Send(config mailtypes.MessageConfig) (any, error) {

	fromHeader := fmt.Sprintf("From: Dropper Notifications <%s>\r\n", config.From)
	toHeader := fmt.Sprintf("To: %s\r\n", config.To)

	msg := []byte(fromHeader + toHeader + config.Subject + config.Mime + "\r\n" + config.Body)
	auth := smtp.PlainAuth("", s.config.SmtpUsername, s.config.SmtpPassword, s.config.SmtpHost)

	if err := smtp.SendMail(s.config.SmtpHost+":"+s.config.SmtpPort, auth, config.From, config.To, msg); err != nil {
		return nil, err
	}

	return nil, nil
}
