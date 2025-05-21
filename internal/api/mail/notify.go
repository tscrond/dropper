package mail

import (
	"fmt"
	"log"

	templates "github.com/tscrond/dropper/internal/mailservice/templates"
	mailtypes "github.com/tscrond/dropper/internal/mailservice/types"
)

type Notifier struct {
	emailSender mailtypes.EmailSender
}

func NewMailNotifier(es mailtypes.EmailSender) Notifier {
	return Notifier{
		emailSender: es,
	}
}

func (n *Notifier) SendSharingNotification(sharedByUser, emailTo, expiryDate string, files []mailtypes.FileInfo) error {

	from := "noreply@dropper-app.win"
	to := []string{emailTo}
	subject := fmt.Sprintf("Subject: New File Transfer from %s", sharedByUser)
	mime := "\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"

	messageConfig := mailtypes.MessageConfig{
		From:    from,
		To:      to,
		Subject: subject,
		Mime:    mime,
	}

	htmlBody, err := templates.RenderMailTemplate("sharing", mailtypes.MailData{
		Files:       files,
		SenderEmail: sharedByUser,
		ExpiryDate:  expiryDate,
	})

	if err != nil {
		log.Println(err)
		return err
	}

	messageConfig.Body = htmlBody

	output, err := n.emailSender.Send(messageConfig)
	if err != nil {
		log.Println("Something went wrong while sending email: ", err)
		return err
	}

	log.Println("Mail sent successfully!", output)

	return nil
}
