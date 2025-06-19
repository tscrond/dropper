package api

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/tscrond/dropper/internal/filedata"
	"github.com/tscrond/dropper/internal/userdata"
	pkg "github.com/tscrond/dropper/pkg"
)

func (s *APIServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", "")
		return
	}

	verifiedUserData, ok := r.Context().Value(userdata.VerifiedUserContextKey).(*userdata.VerifiedUserInfo)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "failed_to_retrieve_user_data", "")
		return
	}

	authorizedUserData, ok := r.Context().Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "failed_to_retrieve_user_data", "")
		return
	}
	// fmt.Println("Authorized User:", authorizedUserData)

	// Get file from request
	file, header, err := r.FormFile("file")
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "failed_parsing_files", "")
		log.Println(err)
		return
	}
	defer file.Close()

	// Create fileData object
	fileData := filedata.NewFileData(file, header)
	if fileData == nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "invalid_file_data", "")
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, userdata.VerifiedUserContextKey, verifiedUserData)
	ctx = context.WithValue(ctx, userdata.AuthorizedUserContextKey, authorizedUserData)

	if err := s.bucketHandler.SendFileToBucket(ctx, fileData); err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "bucket_upload_failed", "")
		return
	}

	msg := fmt.Sprintf("Files uploaded successfully: %+v\n", fileData.RequestHeaders.Filename)

	pkg.WriteJSONResponse(w, http.StatusOK, "", msg)
}
