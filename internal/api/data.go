package api

import (
	"fmt"
	"net/http"

	"github.com/tscrond/dropper/internal/userdata"
)

func (s *APIServer) getUserData(w http.ResponseWriter, r *http.Request) {

	userData, ok := r.Context().Value(userdata.UserContextKey).(*userdata.VerifiedUserInfo)
	fmt.Println(userData)
	if !ok {
		JSON(w, map[string]interface{}{
			"response": "access_denied",
			"code":     http.StatusForbidden,
		})
		return
	}

	response := map[string]interface{}{
		"response":  "ok",
		"user_data": userData,
	}

	JSON(w, response)
}
