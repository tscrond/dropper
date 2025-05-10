package api

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"maps"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
)

func (s *APIServer) downloadThroughProxyPersonal(w http.ResponseWriter, r *http.Request) {
	// 0*. generate the access token (include token in shares db table - /share endpoint)
	// * this step is done in /share endpoint exclusively
	// 1. take in token as a query parameter or path
	// 2. check if user is authorized
	// 3. check if token exists
	// 4. generate short-lived signed URL
	// 5. stream the file output from signed URL to the response writer
	ctx := r.Context()

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	// parse data of logged in user
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"response": "authorization_failed",
			"code":     http.StatusForbidden,
		})
		return
	}

	token := chi.URLParam(r, "token")

	if token == "" {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "empty_token",
			"code":     http.StatusInternalServerError,
		})
	}

	// 2. check if token exists, if exists, return file ID
	_, err := s.repository.Queries.GetFileIdFromToken(ctx, sql.NullString{Valid: true, String: token})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			JSON(w, map[string]any{
				"response": "file_does_not_exist",
				"code":     http.StatusNotFound,
			})
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]any{
				"response": "token_check_error",
				"code":     http.StatusInternalServerError,
			})
			return
		}
	}

	bucketAndObjectRow, err := s.repository.Queries.GetBucketObjectAndOwnerFromPrivateToken(ctx, sql.NullString{Valid: true, String: token})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "cannot_get_bucket_data",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	if authUserData.Id != bucketAndObjectRow.OwnerGoogleID.String {
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"response": "access_denied",
			"code":     http.StatusForbidden,
		})
		return
	}

	bucket := bucketAndObjectRow.BucketName.String
	object := bucketAndObjectRow.ObjectName

	signedUrl, err := s.bucketHandler.GenerateSignedURL(ctx, bucket, object, time.Now().Add(1*time.Minute))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "cannot_generate_url",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	// fmt.Println(signedUrl)

	// 4. stream the file contents to the writer
	resp, err := http.Get(signedUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "signed_url_fetch_failed",
			"code":     http.StatusInternalServerError,
		})
		return
	}
	defer resp.Body.Close()

	// Copy headers
	maps.Copy(w.Header(), resp.Header)
	w.WriteHeader(http.StatusOK)

	// Stream the body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println("streaming error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "streaming_error",
			"code":     http.StatusInternalServerError,
		})
		return
	}

}

func (s *APIServer) downloadThroughProxy(w http.ResponseWriter, r *http.Request) {
	// download link should look like this: https://<domain_name>/download?token=143adfsadfasd9a9sdf7a89df9
	// or: https://<domain_name>/d/143adfsadfasd9a9sdf7a89df9
	// 0*. generate the access token (include token in shares db table - /share endpoint)
	// * this step is done in /share endpoint exclusively
	// 1. take in token as a query parameter or path
	// 2. check if token exists
	// 3. generate short-lived signed URL
	// 4. stream the file output from signed URL to the response writer
	ctx := r.Context()

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	// 1. parse sharing token from url path
	sharingToken := chi.URLParam(r, "token")
	log.Println(sharingToken)

	// 1.5 check token expiration times
	expiresAt, err := s.repository.Queries.GetTokenExpirationTime(ctx, sharingToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "error_checking_expiration",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	if expiresAt.Before(time.Now()) {
		w.WriteHeader(http.StatusForbidden)
		JSON(w, map[string]any{
			"response": "past_expiration_time_or_does_not_exist",
			"code":     http.StatusForbidden,
		})
		return
	}

	// 2. check if token exists, if exists, return shared file ID
	_, err = s.repository.Queries.GetSharedFileIdFromToken(ctx, sharingToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNotFound)
			JSON(w, map[string]any{
				"response": "token_does_not_exist",
				"code":     http.StatusNotFound,
			})
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			JSON(w, map[string]any{
				"response": "token_check_error",
				"code":     http.StatusInternalServerError,
			})
			return
		}
	}

	// 2.5 get the bucket of shared resource + get the object name
	bucketAndObject, err := s.repository.Queries.GetBucketAndObjectFromToken(ctx, sharingToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "cannot_get_bucket_data",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	// 3. generate signed url
	signedUrl, err := s.bucketHandler.GenerateSignedURL(ctx,
		bucketAndObject.UserBucket.String,
		bucketAndObject.FileName,
		time.Now().Add(time.Minute),
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "signed_url_error",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	// 4. stream the file contents to the writer
	resp, err := http.Get(signedUrl)
	if err != nil || resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "signed_url_fetch_failed",
			"code":     http.StatusInternalServerError,
		})
		return
	}
	defer resp.Body.Close()

	// Copy headers
	maps.Copy(w.Header(), resp.Header)
	w.WriteHeader(http.StatusOK)

	// Stream the body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println("streaming error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "streaming_error",
			"code":     http.StatusInternalServerError,
		})
		return
	}
}

func (s *APIServer) getDataSharedForUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		return
	}

	sharedFor := sql.NullString{Valid: true, String: authUserData.Email}
	filesShared, err := s.repository.Queries.GetFilesSharedWithUser(ctx, sharedFor)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "internal_error",
			"code":     http.StatusInternalServerError,
		})
	}

	filesSharedPrep := prepSharedFilesFormat(filesShared)

	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"response": "ok",
		"code":     http.StatusOK,
		"files":    filesSharedPrep,
	})
}

func (s *APIServer) getDataSharedByUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		return
	}

	sharedFor := sql.NullString{Valid: true, String: authUserData.Email}
	filesShared, err := s.repository.Queries.GetFilesSharedByUser(ctx, sharedFor)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "internal_error",
			"code":     http.StatusInternalServerError,
		})
	}

	filesSharedPrep := prepSharedByFilesFormat(filesShared)

	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"response": "ok",
		"code":     http.StatusOK,
		"files":    filesSharedPrep,
	})
}

// helper function for sharing (converts format of table for JSON output)
func prepSharedFilesFormat(sharedFiles []sqlc.GetFilesSharedWithUserRow) []any {

	var allfiles []any
	for _, sharedFile := range sharedFiles {

		savedData := make(map[string]any)

		savedData["file_id"] = sharedFile.FileID.Int32
		savedData["owner_google_id"] = sharedFile.OwnerGoogleID.String
		savedData["file_name"] = sharedFile.FileName
		savedData["file_type"] = sharedFile.FileType.String
		savedData["md5_checksum"] = sharedFile.Md5Checksum
		savedData["shared_by"] = sharedFile.SharedBy.String
		savedData["shared_for"] = sharedFile.SharedFor.String
		savedData["sharing_token"] = sharedFile.SharingToken
		savedData["expires_at"] = sharedFile.ExpiresAt
		savedData["size"] = sharedFile.Size.Int64

		allfiles = append(allfiles, savedData)
	}

	return allfiles
}

// helper function for sharing
func prepSharedByFilesFormat(sharedFiles []sqlc.GetFilesSharedByUserRow) []any {

	var allfiles []any
	for _, sharedFile := range sharedFiles {

		savedData := make(map[string]any)

		savedData["file_id"] = sharedFile.FileID.Int32
		savedData["owner_google_id"] = sharedFile.OwnerGoogleID.String
		savedData["file_name"] = sharedFile.FileName
		savedData["file_type"] = sharedFile.FileType.String
		savedData["md5_checksum"] = sharedFile.Md5Checksum
		savedData["shared_by"] = sharedFile.SharedBy.String
		savedData["shared_for"] = sharedFile.SharedFor.String
		savedData["sharing_token"] = sharedFile.SharingToken
		savedData["expires_at"] = sharedFile.ExpiresAt
		savedData["size"] = sharedFile.Size.Int64

		allfiles = append(allfiles, savedData)
	}

	return allfiles
}
