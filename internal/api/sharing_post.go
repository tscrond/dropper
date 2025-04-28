package api

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/tscrond/dropper/internal/repo/sqlc"
	"github.com/tscrond/dropper/internal/userdata"
	"github.com/tscrond/dropper/pkg"
)

func (s *APIServer) shareWith(w http.ResponseWriter, r *http.Request) {
	// 0*. generate the access token (include token in shares db table - /share endpoint)
	// * this step is done in /share endpoint exclusively
	// 1. take in token as a query parameter or path
	// 2. check if user is authorized
	// 3. check if token exists
	// 4. generate short-lived signed URL
	// 5. stream the file output from signed URL to the response writer
	ctx := r.Context()

	if r.Method != http.MethodPost {
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
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "not_authorized",
			"code":     http.StatusBadRequest,
		})
		return
	}

	forUser := r.URL.Query().Get("email")
	object := r.URL.Query().Get("object")
	shareDuration := r.URL.Query().Get("duration")

	// calculate expiry time
	expiryTime, err := pkg.CustomParseDuration(shareDuration)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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
		w.WriteHeader(http.StatusNotFound)
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
		w.WriteHeader(http.StatusInternalServerError)
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
