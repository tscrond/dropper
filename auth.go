package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

const (
	key    = "asdfasdfasdf"
	MaxAge = 86400 * 30
	IsProd = false
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

	var jsonResp GoAuth

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

	http.Redirect(w, r, "http://localhost:5173/", http.StatusTemporaryRedirect)
}

type GoAuth struct {
	Id    string `json:"id"`
	Email string `json:"email"`
	// VerifiedEmail string `json:"verified_email"`
	Name       string `json:"name"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture    string `json:"picture"`
	Locale     string `json:"locale"`
}

//	{
//		"azp": "asodfkasdofkao-rasdkfkaosdfpasodfkg.apps.googleusercontent.com",
//		"aud": "asodfkasdofkao-rasdkfkaosdfpasodfkg.apps.googleusercontent.com",
//		"sub": "1065436339302349807",
//		"scope": "https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile openid",
//		"exp": "1111432532",
//		"expires_in": "4432",
//		"email": "abc.bca@gmail.com",
//		"email_verified": "true",
//		"access_type": "offline"
//	  }

type VerifiedUserInfo struct {
	Azp           string `json:"azp"`
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Scope         string `json:"scope"`
	Exp           string `json:"exp"`
	ExpiresIn     string `json:"expires_in"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	AccessType    string `json:"access_type"`
}

func (s *APIServer) AuthMiddleware(next http.Handler) http.Handler {
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

		ctx := context.WithValue(r.Context(), "user", userInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *APIServer) verifyToken(cookieValue string) (bool, *VerifiedUserInfo) {
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

	var userInfo VerifiedUserInfo

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Println("cannot decode user info")
		return false, nil
	}

	return true, &userInfo
}

func (s *APIServer) isValid(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println(cookie.Value)

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
