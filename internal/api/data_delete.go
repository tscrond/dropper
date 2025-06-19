package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
	pkg "github.com/tscrond/dropper/pkg"
)

func (s *APIServer) deleteFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
		return
	}

	object := r.URL.Query().Get("file")
	if object == "" {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
	}

	// parse data of logged in user
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		pkg.WriteJSONResponse(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	bucket := fmt.Sprintf("%s-%s", s.bucketHandler.GetBucketBaseName(), authUserData.Id)

	// dont fail if object does not exist, just report the error
	if err := s.bucketHandler.DeleteObjectFromBucket(ctx, object, bucket); err != nil {
		log.Println("issues deleting object: ", err)
	}

	if err := s.repository.Queries.DeleteFileByNameAndId(ctx, sqlc.DeleteFileByNameAndIdParams{
		OwnerGoogleID: sql.NullString{Valid: true, String: authUserData.Id},
		FileName:      object,
	}); err != nil {
		log.Println("errors deleting file from DB: ", err)
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "delete_file_error", nil)
		return
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "success", map[string]any{
		"file_deleted": object,
	})
}

func (s *APIServer) deleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", nil)
		return
	}

	// parse data of logged in user
	authorizedUserData := ctx.Value(userdata.AuthorizedUserContextKey)
	authUserData, ok := authorizedUserData.(*userdata.AuthorizedUserInfo)
	if !ok {
		log.Println("cannot read authorized user data")
		pkg.WriteJSONResponse(w, http.StatusForbidden, "authorization_failed", nil)
		return
	}

	deletedAccount, err := s.repository.Queries.DeleteAccount(ctx, authUserData.Id)
	if err != nil {
		log.Println("issues deleting object: ", err)
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "authorization_failed", nil)
		return
	}
 
	pkg.WriteJSONResponse(w, http.StatusOK, "success", map[string]any{
		"account_deleted": map[string]any{
			"id":        deletedAccount.GoogleID,
			"email":     deletedAccount.UserEmail,
			"user_name": deletedAccount.UserName.String,
		},
	})

}
