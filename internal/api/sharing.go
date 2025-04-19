package api

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"maps"

	"github.com/go-chi/chi/v5"
	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
	"github.com/tscrond/dropper/pkg"
)

func (s *APIServer) shareWith(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	ctx := r.Context()

	// parse data of logged in user
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		return
	}

	forUser := r.URL.Query().Get("email")
	object := r.URL.Query().Get("object")
	shareDuration := r.URL.Query().Get("duration")

	// calculate expiry time
	expiryTime, err := time.ParseDuration(shareDuration)
	if err != nil {
		JSON(w, map[string]any{
			"response": "invalid_duration",
			"code":     http.StatusBadRequest,
		})
		return
	}

	expiresAt := time.Now().Add(expiryTime)

	// get shared object's attributes (id and checksum)
	sharedObjectData, err := s.repository.Queries.GetFileByOwnerAndName(ctx, sqlc.GetFileByOwnerAndNameParams{
		OwnerGoogleID: sql.NullString{Valid: true, String: authUserData.Id},
		FileName:      object,
	})

	if err != nil {
		JSON(w, map[string]any{
			"response": "file_not_found",
			"code":     http.StatusNotFound,
			"err":      err.Error(),
		})
		return
	}

	generatedToken, _ := pkg.RandToken(32)

	share, err := s.repository.Queries.InsertShare(ctx, sqlc.InsertShareParams{
		SharedBy:     sql.NullString{Valid: true, String: authUserData.Email},
		SharedFor:    sql.NullString{Valid: true, String: forUser},
		FileID:       sql.NullInt32{Valid: true, Int32: sharedObjectData.ID},
		ExpiresAt:    expiresAt,
		SharingToken: generatedToken,
	})

	if err != nil {
		log.Println("error inserting new share entry: ", err)
		JSON(w, map[string]any{
			"response": "insert_share_error",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	JSON(w, map[string]any{
		"response": "ok",
		"code":     http.StatusOK,
		"sharing_info": map[string]any{
			"shared_for":    share.SharedFor.String,
			"shared_by":     share.SharedBy.String,
			"checksum":      sharedObjectData.Md5Checksum,
			"expires_at":    share.ExpiresAt,
			"sharing_token": share.SharingToken,
		},
	})
}

func (s *APIServer) downloadThroughProxy(w http.ResponseWriter, r *http.Request) {
	// todo: create downloading proxy for files controlling access via token or ID
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

	// 4. stream the file

	resp, err := http.Get(signedUrl)
	// if err != nil || resp.StatusCode != http.StatusOK {
	if err != nil {
		log.Println("dupaaaaaaaaaaaa:", err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "signed_url_fetch_failed",
			"code":     http.StatusInternalServerError,
		})
		return
	}
	defer resp.Body.Close()

	// Copy headers (optional, like content-type, content-length)
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
