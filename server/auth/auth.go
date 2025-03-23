package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"chess/env"
	"chess/model"

	"github.com/google/uuid"
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

func NewAuthServer(db *model.Queries, environment *env.Env, path string) *AuthServer {
	server := &AuthServer{
		ServeMux: http.NewServeMux(),
		oAuth2Config: &oauth2.Config{
			ClientID:     environment.OauthClientId,
			ClientSecret: environment.OauthClientSecret,
			RedirectURL:  path + "/auth/callback",
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		stateStore: make(StateStoreMap),
		db:         db,
	}

	server.ServeMux.HandleFunc("/login", server.LoginHandler)
	server.ServeMux.HandleFunc("/logout", server.LogoutHandler)
	server.ServeMux.HandleFunc("/callback", server.CallbackHandler)
	server.ServeMux.HandleFunc("/user", server.UserHandler)

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

func (server *AuthServer) createSession(
	writer http.ResponseWriter,
	ctx context.Context,
	userID uuid.UUID,
	accessToken string,
	refreshToken string,
	expiresAt time.Time,
) (uuid.UUID, error) {
	params := model.CreateSessionParams{
		ID:           uuid.New(),
		UserID:       userID.String(),
		AccessToken:  accessToken,
		RefreshToken: nullString(refreshToken),
		ExpiresAt:    expiresAt,
	}
	dbSessionId, err := server.db.CreateSession(ctx, params)
	if err != nil {
		slog.Error(
			"error creating session",
			slog.Any("error", err),
			slog.Any("params", params),
		)
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return uuid.UUID{}, err
	}
	return dbSessionId, err
}

func (server *AuthServer) LoginHandler(writer http.ResponseWriter, r *http.Request) {
	state, err := generateState()
	if err != nil {
		http.Error(writer, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	server.stateStore[state] = time.Now()

	http.SetCookie(writer, &http.Cookie{
		Name:     cookieKeyState,
		Value:    state,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	url := server.oAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(writer, r, url, http.StatusTemporaryRedirect)
}

const (
	UnsetMaxAge int = -1
	NoMaxage        = 0
)

const (
	CookieKeySession = "session-token"
	cookieKeyUser    = "user"
	cookieKeyState   = "oauth_state"
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

func makeCookie(name, value string, httpOnly bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: httpOnly,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24, // 24 hours
	}
}

func (server *AuthServer) CallbackHandler(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	cookie, err := req.Cookie(cookieKeyState)
	if err != nil {
		http.Error(writer, "State cookie not found", http.StatusBadRequest)
		return
	}

	if req.URL.Query().Get("state") != cookie.Value {
		http.Error(writer, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	timestamp, exists := server.stateStore[cookie.Value]
	if !exists || time.Since(timestamp) > 10*time.Minute {
		http.Error(writer, "State expired or invalid", http.StatusBadRequest)
		slog.Error("State expired or invalid",
			slog.Any("since", time.Since(timestamp)))
		delete(server.stateStore, cookie.Value)
		return
	}
	delete(server.stateStore, cookie.Value)

	http.SetCookie(writer, &http.Cookie{
		Name:     cookieKeyState,
		Path:     "/",
		MaxAge:   UnsetMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	code := req.URL.Query().Get("code")
	token, err := server.oAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		http.Error(writer, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	client := server.oAuth2Config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(writer, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(writer, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	dbUser, err := server.db.GetUserByEmail(ctx, userInfo.Email)
	if err == sql.ErrNoRows {
		dbUser, err = server.db.CreateUser(ctx,
			model.CreateUserParams{
				ID:       uuid.New(),
				Username: nullString(userInfo.Name),
				Email:    userInfo.Email,
			})
		if err != nil {
			slog.Error(
				"an error was returned when creating a new user",
				slog.Any("error", err),
			)
			http.Error(writer, "Failed querying db", http.StatusInternalServerError)
			return
		}
	} else if err != nil {
		slog.Error(
			"a non sql.ErrNoRows err was returned when getting user by email",
			slog.Any("error", err),
		)
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return
	}

	dbSessionId, err := server.createSession(
		writer, ctx,
		dbUser.ID,
		token.AccessToken,
		token.RefreshToken,
		token.Expiry,
	)
	if err != nil {
		// error handled in above function
		return
	}

	userBytes, _ := json.Marshal(userInfo)
	http.SetCookie(writer, makeCookie(cookieKeyUser,
		base64.URLEncoding.EncodeToString(userBytes), false))
	http.SetCookie(writer,
		makeCookie(CookieKeySession, dbSessionId.String(), true))

	// todo redirect
	http.Redirect(writer, req, "http://localhost:3000", http.StatusTemporaryRedirect)
}

func getSessionId(writer http.ResponseWriter, req *http.Request) (uuid.UUID, error) {
	cookie, err := req.Cookie(CookieKeySession)
	if err != nil {
		http.Error(writer, "Session cookie not found", http.StatusBadRequest)
		return uuid.UUID{}, err
	}

	sessionId, err := uuid.Parse(cookie.Value)
	if err != nil {
		http.Error(writer,
			"session id was not able to be parsed into uuid",
			http.StatusInternalServerError)
		return uuid.UUID{}, err
	}
	return sessionId, err
}

func unsetCookie(writer http.ResponseWriter, key string) {
	http.SetCookie(writer, &http.Cookie{
		Name:     key,
		Path:     "/",
		MaxAge:   UnsetMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (server *AuthServer) LogoutHandler(writer http.ResponseWriter, req *http.Request) {
	sessionId, err := getSessionId(writer, req)
	if err != nil {
		return
	}

	unsetCookie(writer, cookieKeyState)
	unsetCookie(writer, CookieKeySession)
	unsetCookie(writer, cookieKeyUser)

	http.Redirect(writer, req, "/", http.StatusTemporaryRedirect)

	ctx := req.Context()
	err = server.db.DeleteSessionsById(ctx, sessionId)
	if err == sql.ErrNoRows {
		return
	} else if err != nil {
		slog.Error(
			"an error occurred while deleting sessions",
			slog.Any("error", err),
		)
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return
	}
}

func (server *AuthServer) RefreshToken(
	token *model.Session,
	writer http.ResponseWriter,
	req *http.Request,
) {
	sessionId, err := getSessionId(writer, req)
	if err != nil {
		return
	}

	ctx := req.Context()
	session, err := server.db.GetSessionById(ctx, sessionId)
	if err != nil {
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return
	}

	if session.ExpiresAt.Before(time.Now()) {
		http.Error(writer, "Session expired", http.StatusUnauthorized)
		return
	}

	newToken, err := server.oAuth2Config.TokenSource(ctx, &oauth2.Token{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken.String,
		Expiry:       session.ExpiresAt,
	}).Token()
	if err != nil {
		slog.Error(
			"error generating token",
			slog.Any("error", err),
		)
		http.Error(writer, "Failed generating token", http.StatusInternalServerError)
		return
	}

	userId, err := uuid.Parse(session.UserID)
	if err != nil {
		http.Error(writer, "Error paring uuid",
			http.StatusInternalServerError)
		slog.Error(
			"uuid parsing error",
			slog.Any("error", err),
			slog.String("uuid", session.UserID),
		)
		return
	}

	dbSession, err := server.createSession(
		writer, ctx,
		userId,
		newToken.AccessToken,
		newToken.RefreshToken,
		newToken.Expiry,
	)
	if err != nil {
		return
	}

	http.SetCookie(writer,
		makeCookie(CookieKeySession, dbSession.String(), true))
	http.Redirect(writer, req, "/", http.StatusTemporaryRedirect)
}

func (server *AuthServer) RefreshHandler(writer http.ResponseWriter, req *http.Request) {
	sessionId, err := getSessionId(writer, req)
	if err != nil {
		return
	}

	ctx := req.Context()
	session, err := server.db.GetSessionById(ctx, sessionId)
	if err != nil {
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return
	}

	if session.ExpiresAt.Before(time.Now()) {
		http.Error(writer, "Session expired", http.StatusUnauthorized)
		return
	}

	newToken, err := server.oAuth2Config.TokenSource(ctx, &oauth2.Token{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken.String,
		Expiry:       session.ExpiresAt,
	}).Token()
	if err != nil {
		slog.Error(
			"error generating token",
			slog.Any("error", err),
		)
		http.Error(writer, "Failed generating token", http.StatusInternalServerError)
		return
	}

	userId, err := uuid.Parse(session.UserID)
	if err != nil {
		http.Error(writer, "Error paring uuid",
			http.StatusInternalServerError)
		slog.Error(
			"uuid parsing error",
			slog.Any("error", err),
			slog.String("uuid", session.UserID),
		)
		return
	}

	dbSession, err := server.createSession(
		writer, ctx,
		userId,
		newToken.AccessToken,
		newToken.RefreshToken,
		newToken.Expiry,
	)
	if err != nil {
		return
	}

	http.SetCookie(writer,
		makeCookie(CookieKeySession, dbSession.String(), true))
	http.Redirect(writer, req, "/", http.StatusTemporaryRedirect)
}

func (server *AuthServer) UserHandler(writer http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	sessionId, err := getSessionId(writer, req)
	if err != nil {
		return
	}

	sessionAndUser, err := server.db.GetSessionByIdAndUser(ctx, sessionId)
	if err != nil {
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(sessionAndUser)
	if err != nil {
		return
	}

	writer.Write(body)
	writer.WriteHeader(http.StatusOK)
}

func (server *AuthServer) GetUserSession(
	ctx context.Context,
	writer http.ResponseWriter,
	req *http.Request,
) (model.GetSessionByIdAndUserRow, error) {
	sessionId, err := getSessionId(writer, req)
	if err != nil {
		return model.GetSessionByIdAndUserRow{}, err
	}

	sessionAndUser, err := server.db.GetSessionByIdAndUser(ctx, sessionId)
	if err != nil {
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return model.GetSessionByIdAndUserRow{}, err
	}

	return sessionAndUser, err
}

func (server *AuthServer) IsAuthenticated(
	ctx context.Context, writer http.ResponseWriter, req *http.Request,
) (bool, error) {
	sessionId, err := getSessionId(writer, req)
	if err != nil {
		return false, err
	}

	_, err = server.db.GetSessionExists(ctx, sessionId)
	if err == sql.ErrNoRows {
		http.Error(writer, "No db session found", http.StatusUnauthorized)
		return false, err
	} else if err != nil {
		slog.Error(
			"error retrieving session",
			slog.Any("error", err),
		)
		http.Error(writer, "Failed querying db", http.StatusInternalServerError)
		return false, err
	}

	return true, nil
}
