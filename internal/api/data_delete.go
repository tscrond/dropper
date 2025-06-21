package api

import (
	"database/sql"
	"encoding/json"
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
	bucketName := pkg.GetUserBucketName(s.bucketHandler.GetBucketBaseName(), authUserData.Id)

	fullResponse := map[string]any{}
	fullResponse["bucket"] = map[string]any{
		"name":    bucketName,
		"deleted": false,
	}

	type DeleteAccountRequest struct {
		DeleteUserData bool `json:"delete_user_data"`
	}

	var req DeleteAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", "")
		return
	}

	if req.DeleteUserData {
		if err := s.bucketHandler.DeleteBucket(ctx, bucketName); err != nil {
			log.Printf("failed to delete bucket %s err: %s\n", bucketName, err)
			fullResponse["bucket"] = map[string]any{
				"name":    bucketName,
				"deleted": false,
			}
		}

		fullResponse["bucket"] = map[string]any{
			"name":    bucketName,
			"deleted": true,
		}
	}

	deletedAccount, err := s.repository.Queries.DeleteAccount(ctx, authUserData.Id)
	if err != nil {
		log.Println("issues deleting object: ", err)
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "authorization_failed", nil)
		return
	}

	fullResponse["account_deleted"] = map[string]any{
		"id":        deletedAccount.GoogleID,
		"email":     deletedAccount.UserEmail,
		"user_name": deletedAccount.UserName.String,
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "success", fullResponse)

}
