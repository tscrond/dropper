package api

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/tscrond/dropper/internal/filedata"
	"github.com/tscrond/dropper/internal/userdata"
)

func (s *APIServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSON(w, map[string]interface{}{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	verifiedUserData, ok := r.Context().Value(userdata.VerifiedUserContextKey).(*userdata.VerifiedUserInfo)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"status":   http.StatusInternalServerError,
			"response": "Failed to retrieve verified user data",
		})
		return
	}

	authorizedUserData, ok := r.Context().Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"status":   http.StatusForbidden,
			"response": "Failed to retrieve authorized user data",
		})
		return
	}
	// fmt.Println("Authorized User:", authorizedUserData)

	// Get file from request
	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"status":   http.StatusBadRequest,
			"response": "Failed to parse file from request",
		})
		log.Println(err)
		return
	}
	defer file.Close()

	// Create fileData object
	fileData := filedata.NewFileData(file, header)
	if fileData == nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"status":   http.StatusInternalServerError,
			"response": "Invalid file data",
		})
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, userdata.VerifiedUserContextKey, verifiedUserData)
	ctx = context.WithValue(ctx, userdata.AuthorizedUserContextKey, authorizedUserData)

	if err := s.bucketHandler.SendFileToBucket(ctx, fileData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"status":   http.StatusInternalServerError,
			"response": "Failed to send file to bucket",
		})
		return
	}

	msg := fmt.Sprintf("Files uploaded successfully: %+v\n", fileData.RequestHeaders.Filename)
	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"status":   http.StatusOK,
		"response": msg,
	})
}
