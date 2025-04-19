package api

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
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

	share, err := s.repository.Queries.InsertShare(ctx, sqlc.InsertShareParams{
		SharedBy:  sql.NullString{Valid: true, String: authUserData.Email},
		SharedFor: sql.NullString{Valid: true, String: forUser},
		FileID:    sql.NullInt32{Valid: true, Int32: sharedObjectData.ID},
		ExpiresAt: expiresAt,
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
			"shared_for": share.SharedFor.String,
			"shared_by":  share.SharedBy.String,
			"checksum":   sharedObjectData.Md5Checksum,
			"expires_at": share.ExpiresAt,
		},
	})
}

func (s *APIServer) downloadThroughProxy(w http.ResponseWriter, r *http.Request) {
	// todo: create downloading proxy for files controlling access via token or ID
	// download link should look like this: https://<domain_name>/download?token=143adfsadfasd9a9sdf7a89df9
	// or: https://<domain_name>/d/143adfsadfasd9a9sdf7a89df9
}
