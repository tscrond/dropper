package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
)

func (s *APIServer) deleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
		return
	}

	object := r.URL.Query().Get("file")
	if object == "" {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "bad_request",
			"code":     http.StatusBadRequest,
		})
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

	bucket := fmt.Sprintf("%s-%s", s.bucketHandler.GetBucketBaseName(), authUserData.Id)

	if err := s.bucketHandler.DeleteObjectFromBucket(object, bucket); err != nil {
		log.Println("issues deleting object: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "authorization_failed",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	if err := s.repository.Queries.DeleteFileByNameAndId(ctx, sqlc.DeleteFileByNameAndIdParams{
		OwnerGoogleID: sql.NullString{Valid: true, String: authUserData.Id},
		FileName:      object,
	}); err != nil {
		log.Println("errors deleting file from DB: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "delete_file_error",
			"code":     http.StatusInternalServerError,
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"response":     "success",
		"code":         http.StatusOK,
		"file_deleted": object,
	})

}

func (s *APIServer) deleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
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

	deletedAccount, err := s.repository.Queries.DeleteAccount(ctx, authUserData.Id)
	if err != nil {
		log.Println("issues deleting object: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		JSON(w, map[string]any{
			"response": "authorization_failed",
			"code":     http.StatusInternalServerError,
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	JSON(w, map[string]any{
		"response": "success",
		"code":     http.StatusOK,
		"account_deleted": map[string]any{
			"id":        deletedAccount.GoogleID,
			"email":     deletedAccount.UserEmail,
			"user_name": deletedAccount.UserName.String,
		},
	})
}
