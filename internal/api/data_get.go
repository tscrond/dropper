package api

import (
	"net/http"

	"github.com/tscrond/dropper/internal/userdata"
	"github.com/tscrond/dropper/pkg"
)

func (s *APIServer) getUserData(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "", "bad_request")
		return
	}

	userData, ok := r.Context().Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	// fmt.Println(userData)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "Access Denied", map[string]any{
			"user_data": nil,
		})
		return
	}

	response := map[string]any{
		"user_data": userData,
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "", response)
}

func (s *APIServer) getUserBucketData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "", "bad_request")
		return
	}

	userData, ok := r.Context().Value(userdata.AuthorizedUserContextKey).(*userdata.AuthorizedUserInfo)
	// fmt.Println(userData)
	if !ok {
		pkg.WriteJSONResponse(w, http.StatusForbidden, "access_denied", map[string]any{
			"user_data": nil,
		})
		return
	}

	bucketData, err := s.bucketHandler.GetUserBucketData(ctx, userData.Id)
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "internal_error", map[string]any{
			"bucket_data": nil,
		})
		return
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "internal_error", map[string]any{
		"bucket_data": bucketData,
	})
}

func (s *APIServer) getUserPrivateFileByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		pkg.WriteJSONResponse(w, http.StatusBadRequest, "bad_request", "")
		return
	}

	ctx := r.Context()

	fileName := r.URL.Query().Get("file")

	downloadToken, err := s.repository.Queries.GetPrivateDownloadTokenByFileName(ctx, fileName)
	if err != nil {
		pkg.WriteJSONResponse(w, http.StatusInternalServerError, "internal_error", "")
		return
	}

	pkg.WriteJSONResponse(w, http.StatusOK, "", map[string]any{
		"private_download_token": downloadToken.String,
	})
}
