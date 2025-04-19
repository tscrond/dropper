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
		http.Error(w, "Failed to retrieve verified user data", http.StatusForbidden)
		return
	}

	authorizedUserData, ok := r.Context().Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		http.Error(w, "Failed to retrieve authorized user data", http.StatusForbidden)
		return
	}
	// fmt.Println("Authorized User:", authorizedUserData)

	// Get file from request
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to parse file from request", http.StatusBadRequest)
		log.Println(err)
		return
	}
	defer file.Close()

	// Create fileData object
	fileData := filedata.NewFileData(file, header)
	if fileData == nil {
		http.Error(w, "Invalid file data", http.StatusInternalServerError)
		log.Println("Invalid file data")
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, userdata.VerifiedUserContextKey, verifiedUserData)
	ctx = context.WithValue(ctx, userdata.AuthorizedUserContextKey, authorizedUserData)

	if err := s.bucketHandler.SendFileToBucket(ctx, fileData); err != nil {
		http.Error(w, "Failed to send file to bucket:", http.StatusInternalServerError)
		// fmt.Fprintf(w, "error uploading files: %+v\n", err)
	}

	fmt.Fprintf(w, "Files uploaded successfully: %+v\n", fileData.RequestHeaders.Filename)
}
