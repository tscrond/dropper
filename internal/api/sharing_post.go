package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	mail "github.com/tscrond/dropper/internal/api/mail"
	mailtypes "github.com/tscrond/dropper/internal/mailservice/types"
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

	type ShareRequest struct {
		ForUser  string   `json:"email"`
		Objects  []string `json:"objects"`
		Duration string   `json:"duration"`
	}

	var req ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid_json", http.StatusBadRequest)
		return
	}

	// calculate expiry time
	expiryTime, err := pkg.CustomParseDuration(req.Duration)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		JSON(w, map[string]any{
			"response": "invalid_duration",
			"code":     http.StatusBadRequest,
		})
		return
	}

	expiresAt := time.Now().Add(expiryTime)

	sharingInfos := make([]map[string]any, 0)
	filesForMail := make([]mailtypes.FileInfo, 0)

	for _, objectName := range req.Objects {
		// get shared object's attributes (id and checksum)
		sharedObjectData, err := s.repository.Queries.GetFileByOwnerAndName(ctx, sqlc.GetFileByOwnerAndNameParams{
			OwnerGoogleID: sql.NullString{Valid: true, String: authUserData.Id},
			FileName:      objectName,
		})

		if err != nil {
			log.Println("error getting object data", err)
			continue
		}

		generatedToken, _ := pkg.RandToken(32)

		share, err := s.repository.Queries.InsertShare(ctx, sqlc.InsertShareParams{
			SharedBy:     sql.NullString{Valid: true, String: authUserData.Email},
			SharedFor:    sql.NullString{Valid: true, String: req.ForUser},
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
		sharingInfos = append(sharingInfos, map[string]any{
			"file":          objectName,
			"shared_for":    share.SharedFor.String,
			"shared_by":     share.SharedBy.String,
			"checksum":      sharedObjectData.Md5Checksum,
			"expires_at":    share.ExpiresAt,
			"sharing_token": share.SharingToken,
		})
		filesForMail = append(filesForMail, mailtypes.FileInfo{
			FileName:    objectName,
			DownloadURL: fmt.Sprintf("%s/d/%s?mode=inline", s.backendConfig.BackendEndpoint, share.SharingToken),
		})
	}

	mailNotifier := mail.NewMailNotifier(s.emailSender)

	mailErr := mailNotifier.SendSharingNotification(
		authUserData.Email,
		req.ForUser,
		expiresAt.Format("2006-01-02 15:04"),
		filesForMail,
	)

	mailStatus := "sent"
	if mailErr != nil {
		log.Println("issues sending email notification: ", mailErr)
		mailStatus = "failed"
	}

	JSON(w, map[string]any{
		"response":            "ok",
		"code":                http.StatusOK,
		"sharing_info":        sharingInfos,
		"notification_status": mailStatus,
	})
}
