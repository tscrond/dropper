package api

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/tscrond/dropper/internal/filedata"
	"github.com/tscrond/dropper/internal/pathutil"
	"github.com/tscrond/dropper/internal/repo/sqlc"
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

	// Get file from request
	file, header, err := r.FormFile("file")
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "failed_parsing_files", "")
		log.Println(err)
		return
	}
	defer file.Close()

	// Resolve canonical path: folder/filename or Main/filename
	folder := r.FormValue("folder")
	var canonicalPath string
	if folder != "" {
		canonicalPath = folder + "/" + header.Filename
	} else {
		canonicalPath = pathutil.WithMainPrefix(header.Filename)
	}

	if err := pathutil.Validate(canonicalPath); err != nil {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "invalid_path", err.Error())
		return
	}

	ctx := r.Context()
	ctx = context.WithValue(ctx, userdata.VerifiedUserContextKey, verifiedUserData)
	ctx = context.WithValue(ctx, userdata.AuthorizedUserContextKey, authorizedUserData)

	// Reject duplicate paths
	_, dupErr := s.repository.Queries.GetFileByOwnerAndName(ctx, sqlc.GetFileByOwnerAndNameParams{
		OwnerGoogleID: sql.NullString{Valid: true, String: authorizedUserData.Id},
		FileName:      canonicalPath,
	})
	if dupErr == nil {
		pkg.WriteJSONResponse(w, http.StatusConflict, "file_already_exists", "")
		return
	}

	// Override filename with canonical path so storage drivers use it as the object key
	header.Filename = canonicalPath

	// Create fileData object
	fileData := filedata.NewFileData(file, header)
	if fileData == nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "invalid_file_data", "")
		return
	}

	if err := s.bucketHandler.SendFileToBucket(ctx, fileData); err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "bucket_upload_failed", "")
		return
	}

	msg := fmt.Sprintf("Files uploaded successfully: %+v\n", canonicalPath)

	pkg.WriteJSONResponse(w, http.StatusOK, "", msg)
}
