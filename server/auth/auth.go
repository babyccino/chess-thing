package auth

import (
	"chess/model"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
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
	db           *model.Queries
}

func NewAuthServer(db *model.Queries) *AuthServer {
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

func (server *AuthServer) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	server.stateStore[state] = time.Now()

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	url := server.oAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

const (
	UnsetMaxAge int = -1
	NoMaxage        = 0
)

type GoogleUserInfo struct {
	Sub           string `json:"sub"`            // Unique identifier for the user
	Name          string `json:"name"`           // Full name of the user
	GivenName     string `json:"given_name"`     // First name of the user
	FamilyName    string `json:"family_name"`    // Last name of the user
	Picture       string `json:"picture"`        // URL of the user's profile picture
	Email         string `json:"email"`          // User's email address
	EmailVerified bool   `json:"email_verified"` // Whether the email is verified
	Locale        string `json:"locale"`         // User's preferred locale
}

func nullString(str string) sql.NullString {
	return sql.NullString{String: str, Valid: true}
}

func makeCookie(name, value string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24, // 24 hours
	}
}

func (server *AuthServer) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "State cookie not found", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != cookie.Value {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	timestamp, exists := server.stateStore[cookie.Value]
	if !exists || time.Since(timestamp) > 10*time.Minute {
		http.Error(w, "State expired or invalid", http.StatusBadRequest)
		delete(server.stateStore, cookie.Value)
		return
	}
	delete(server.stateStore, cookie.Value)

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Path:     "/",
		MaxAge:   UnsetMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	code := r.URL.Query().Get("code")
	token, err := server.oAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := server.oAuth2Config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	dbUser, err := server.db.GetUserByEmail(ctx, userInfo.Email)
	if err == sql.ErrNoRows {
		dbUser, err = server.db.CreateUser(ctx,
			model.CreateUserParams{
				Username: nullString(userInfo.Name),
				Email:    userInfo.Email,
			})
	} else if err != nil {
		slog.Error(
			"a non sql.ErrNoRows err was returned when getting user by email",
			slog.Any("error", err),
		)
		http.Error(w, "Failed querying db", http.StatusInternalServerError)
		return
	}

	dbSession, err := server.db.CreateSession(ctx, model.CreateSessionParams{
		UserID:       dbUser.ID,
		AccessToken:  token.AccessToken,
		RefreshToken: nullString(token.RefreshToken),
		ExpiresAt:    token.Expiry,
	})
	if err != nil {
		slog.Error(
			"error creating session",
			slog.Any("error", err),
		)
		http.Error(w, "Failed querying db", http.StatusInternalServerError)
		return
	}

	userBytes, _ := json.Marshal(userInfo)
	http.SetCookie(w, makeCookie("user", base64.URLEncoding.EncodeToString(userBytes)))
	http.SetCookie(w, makeCookie("session-token", dbSession.ID))

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
