package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/tscrond/dropper/internal/userdata"
	"golang.org/x/oauth2"
)

const (
	IsProd = true
)

func (s *APIServer) oauthHandler(w http.ResponseWriter, r *http.Request) {
	url := s.OAuthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *APIServer) authCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")

	t, err := s.OAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// fmt.Println(t)

	// client := s.OAuthConfig.Client(context.Background(), t)
	client := s.OAuthConfig.Client(context.Background(), t)

	// Getting the user public details from google API endpoint
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	var jsonResp userdata.AuthorizedUserInfo

	// Reading the JSON body using JSON decoder
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("%+v", jsonResp)

	// Store user information in a session (cookie)
	sessionCookie := &http.Cookie{
		Name:     "session_id",
		Value:    fmt.Sprintf("%s", t.AccessToken),
		HttpOnly: true,
		Secure:   IsProd,
		Path:     "/",
	}
	http.SetCookie(w, sessionCookie)

	http.Redirect(w, r, s.frontendEndpoint, http.StatusTemporaryRedirect)
}

func (s *APIServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		fmt.Println(cookie)
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}

		valid, userInfo := s.verifyToken(cookie.Value)
		if !valid {
			http.Error(w, "Unauthorized (invalid or expired session)", http.StatusForbidden)
			return
		}

		// var accessToken string
		// userInfo, err := s.fetchUserInfo(accessToken)
		fmt.Println(userInfo)

		ctx := context.WithValue(r.Context(), userdata.UserContextKey, userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *APIServer) verifyToken(cookieValue string) (bool, *userdata.VerifiedUserInfo) {
	resp, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s", cookieValue))
	if err != nil {
		log.Println(err)
		return false, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("status code not OK")
		return false, nil
	}

	var userInfo userdata.VerifiedUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Println("cannot decode user info")
		return false, nil
	}

	return true, &userInfo
}

// Revoke OAuth2 token and expire session cookie
func (s *APIServer) logout(w http.ResponseWriter, r *http.Request) {
	// Check if session_id cookie exists
	cookie, err := r.Cookie("session_id")
	if err != nil {
		JSON(w, map[string]interface{}{
			"response":          "cookie_not_found",
			"code":              http.StatusNotFound,
			"logout_successful": true,
		})
		return
	}

	// Prepare request to revoke OAuth2 token
	revokeURL := "https://oauth2.googleapis.com/revoke"
	formData := url.Values{}
	formData.Set("token", cookie.Value)

	req, err := http.NewRequest("POST", revokeURL, nil)
	if err != nil {
		JSON(w, map[string]interface{}{
			"response":          "internal_server_error",
			"code":              http.StatusInternalServerError,
			"logout_successful": false,
		})
		return
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.URL.RawQuery = formData.Encode() // Send token in body

	// Send request
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		JSON(w, map[string]interface{}{
			"response":          "internal_server_error",
			"code":              http.StatusInternalServerError,
			"logout_successful": false,
		})
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		JSON(w, map[string]interface{}{
			"response":          "failed_to_revoke_token",
			"code":              resp.StatusCode,
			"logout_successful": false,
		})
		return
	}

	// Expire session cookie
	expiredCookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0), // Expire immediately
		MaxAge:   -1,              // Remove from browser
	}

	http.SetCookie(w, expiredCookie)

	// Return success response
	JSON(w, map[string]interface{}{
		"response":          "session_invalidated",
		"code":              http.StatusOK,
		"logout_successful": true,
	})
}

func (s *APIServer) isValid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		JSON(w, map[string]interface{}{
			"response":      "bad_request",
			"code":          http.StatusBadRequest,
			"authenticated": false,
		})
		return
	}
	cookie, err := r.Cookie("session_id")
	if err != nil || cookie.Value == "" {
		response := map[string]interface{}{
			"response":      "access_denied",
			"code":          http.StatusForbidden,
			"authenticated": false,
		}
		JSON(w, response)
		return
	}

	// fmt.Println(cookie.Value)

	valid, userInfo := s.verifyToken(cookie.Value)
	if !valid {
		response := map[string]interface{}{
			"response":      "access_denied",
			"code":          http.StatusForbidden,
			"authenticated": false,
		}
		JSON(w, response)
		return
	}

	response := map[string]interface{}{
		"response":      "access_granted",
		"code":          http.StatusOK,
		"authenticated": true,
		"user_info":     userInfo,
	}
	JSON(w, response)
}

func JSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
}
