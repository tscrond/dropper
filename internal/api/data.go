package api

import (
	"fmt"
	"net/http"

	"github.com/tscrond/dropper/internal/userdata"
)

func (s *APIServer) getUserData(w http.ResponseWriter, r *http.Request) {

	userData, ok := r.Context().Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	fmt.Println(userData)
	if !ok {
		JSON(w, map[string]interface{}{
			"response":  "access_denied",
			"code":      http.StatusForbidden,
			"user_data": nil,
		})
		return
	}

	response := map[string]interface{}{
		"response":  "ok",
		"code":      http.StatusOK,
		"user_data": userData,
	}

	JSON(w, response)
}

func (s *APIServer) getUserBucketData(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		JSON(w, map[string]any{
			"response":    "bad_request",
			"code":        http.StatusBadRequest,
			"bucket_data": nil,
		})
		return
	}
	ctx := r.Context()

	userData, ok := r.Context().Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	fmt.Println(userData)
	if !ok {
		JSON(w, map[string]any{
			"response":  "access_denied",
			"code":      http.StatusForbidden,
			"user_data": nil,
		})
		return
	}

	bucketData, err := s.bucketHandler.GetUserBucketData(ctx, userData.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response":    "internal_error",
			"code":        http.StatusInternalServerError,
			"bucket_data": nil,
		})
		return
	}

	JSON(w, map[string]any{
		"response":    "ok",
		"code":        http.StatusOK,
		"bucket_data": bucketData,
	})
}
