package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type StateStoreMap map[string]time.Time

type AuthServer struct {
	ServeMux     *http.ServeMux
	oAuth2Config *oauth2.Config
	stateStore   StateStoreMap
	db           *sql.DB
}

func NewAuthServer(db *sql.DB) *AuthServer {
	server := &AuthServer{
		ServeMux: http.NewServeMux(),
		oAuth2Config: &oauth2.Config{
			ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
			ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
			RedirectURL:  "http://localhost:8080/callback",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		stateStore: make(StateStoreMap),
	}

	server.ServeMux.HandleFunc("/login", server.LoginHandler)
	server.ServeMux.HandleFunc("/callback", server.CallbackHandler)
	server.ServeMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome! <a href='/login'>Login</a>")
	})

	return server
}

func (server *AuthServer) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	server.ServeMux.ServeHTTP(writer, req)
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (c *AuthServer) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	c.stateStore[state] = time.Now()

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	url := c.oAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (c *AuthServer) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != cookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	timestamp, exists := c.stateStore[cookie.Value]
	if !exists || time.Since(timestamp) > 10*time.Minute {
		http.Error(w, "State expired or invalid", http.StatusBadRequest)
		delete(c.stateStore, cookie.Value)
		return
	}
	delete(c.stateStore, cookie.Value)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")
	token, err := c.oAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := c.oAuth2Config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	session, _ := json.Marshal(userInfo)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    base64.URLEncoding.EncodeToString(session),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600 * 24, // 24 hours
	})

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
