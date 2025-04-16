package api

import (
	"fmt"
	"net/http"
)

func (s *APIServer) shareWith(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSON(w, map[string]any{
			"response":      "bad_request",
			"code":          http.StatusBadRequest,
			"authenticated": false,
		})
		return
	}

	ctx := r.Context()

	bucket, err := s.bucketHandler.GetUserBucketName(ctx)
	if err != nil {
		JSON(w, map[string]any{
			"response":      "internal_error",
			"code":          http.StatusInternalServerError,
			"authenticated": false,
		})
		return
	}

	forUser := r.URL.Query().Get("email")
	fmt.Println(forUser)
	object := r.URL.Query().Get("object")

	fmt.Println(object)

	signedURL, err := s.bucketHandler.GenerateSignedURL(ctx, bucket, object)
	if err != nil {
		JSON(w, map[string]any{
			"response":      "internal_error",
			"code":          http.StatusInternalServerError,
			"authenticated": false,
		})
		return
	}

	JSON(w, map[string]any{
		"response":   "ok",
		"code":       http.StatusOK,
		"signed_url": signedURL,
	})
}
