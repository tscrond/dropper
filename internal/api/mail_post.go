package api

import (
	"log"
	"net/http"

	"github.com/tscrond/dropper/internal/mailservice/types"
)

func (s *APIServer) sendEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	output, err := s.emailSender.Send(types.MessageConfig{
		From:    "notifier@dropper-app.win",
		To:      []string{"bobak.labs@gmail.com"},
		Subject: "cze",
		Body:    "<h1>eeeeeeeo</h1>",
	})
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"response": output,
		"code":     http.StatusOK,
	})
}
