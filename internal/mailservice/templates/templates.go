package templates

import (
	"bytes"
	"errors"
	"html/template"
	"log"
	"path/filepath"

	types "github.com/tscrond/dropper/internal/mailservice/types"
)

func RenderMailTemplate(templateType string, emailData types.MailData) (string, error) {
	switch templateType {
	case "sharing":
		tmpl, err := template.ParseFiles(filepath.Clean("./internal/mailservice/templates/share.html"))
		if err != nil {
			log.Println(err)
			return "", err
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, emailData); err != nil {
			return "", err
		}

		templatedBody := buf.String()
		return templatedBody, nil

	default:
		return "", errors.New("no available template")
	}
}
