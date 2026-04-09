package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/http"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"

	"mangafoxy/internal/config"
)

const sessionName = "mangafoxy-session"

// SessionUser holds the authenticated user's identity.
type SessionUser struct {
	Email string
	Name  string
}

// AuthState holds all OAuth2/OIDC state needed across handlers.
type AuthState struct {
	provider *gooidc.Provider
	oauth2   oauth2.Config
	store    *sessions.CookieStore
}

// InitAuth initialises the OIDC provider and OAuth2 config via OIDC discovery.
// Returns nil (with a log warning) if Okta env vars are not set, so the app
// still starts without auth configured.
func InitAuth(cfg *config.Config) *AuthState {
	if cfg.OktaIssuer == "" || cfg.OktaClientID == "" {
		log.Println("Auth: OKTA_ISSUER or OKTA_CLIENT_ID not set — auth disabled")
		return nil
	}

	provider, err := gooidc.NewProvider(context.Background(), cfg.OktaIssuer)
	if err != nil {
		log.Fatalf("Auth: OIDC provider init failed: %v", err)
	}

	oa2 := oauth2.Config{
		ClientID:     cfg.OktaClientID,
		ClientSecret: cfg.OktaClientSecret,
		RedirectURL:  cfg.OktaRedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{gooidc.ScopeOpenID, "profile", "email"},
	}

	store := sessions.NewCookieStore([]byte(cfg.SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   cfg.Env == "prod",
		SameSite: http.SameSiteLaxMode,
	}

	return &AuthState{
		provider: provider,
		oauth2:   oa2,
		store:    store,
	}
}

// Login generates PKCE + state, stores them in the session, and redirects to Okta.
func Login(auth *AuthState) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			c.String(http.StatusServiceUnavailable, "Auth not configured")
			return
		}

		// PKCE code verifier (43–128 chars of random base64url)
		verifier := randomBase64URL(32)
		// code_challenge = BASE64URL(SHA256(verifier))
		challenge := pkceChallenge(verifier)
		// state
		state := randomBase64URL(16)

		sess, _ := auth.store.Get(c.Request, sessionName)
		sess.Values["pkce_verifier"] = verifier
		sess.Values["oauth_state"] = state
		if err := sess.Save(c.Request, c.Writer); err != nil {
			log.Printf("Login: session save error: %v", err)
			c.String(http.StatusInternalServerError, "session error")
			return
		}

		authURL := auth.oauth2.AuthCodeURL(state,
			oauth2.SetAuthURLParam("code_challenge", challenge),
			oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		)
		c.Redirect(http.StatusFound, authURL)
	}
}

// Callback validates state, exchanges the code+verifier for tokens, verifies the
// ID token, saves identity to session, and redirects to /.
func Callback(auth *AuthState) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			c.String(http.StatusServiceUnavailable, "Auth not configured")
			return
		}

		sess, err := auth.store.Get(c.Request, sessionName)
		if err != nil {
			c.String(http.StatusBadRequest, "session error")
			return
		}

		// Validate state
		savedState, _ := sess.Values["oauth_state"].(string)
		if savedState == "" || savedState != c.Query("state") {
			c.String(http.StatusBadRequest, "invalid state")
			return
		}

		// Retrieve PKCE verifier
		verifier, _ := sess.Values["pkce_verifier"].(string)
		if verifier == "" {
			c.String(http.StatusBadRequest, "missing code verifier")
			return
		}

		// Exchange code for tokens
		token, err := auth.oauth2.Exchange(c.Request.Context(), c.Query("code"),
			oauth2.SetAuthURLParam("code_verifier", verifier),
		)
		if err != nil {
			log.Printf("Callback: token exchange error: %v", err)
			c.String(http.StatusUnauthorized, "token exchange failed")
			return
		}

		// Verify ID token
		rawID, ok := token.Extra("id_token").(string)
		if !ok {
			c.String(http.StatusUnauthorized, "missing id_token")
			return
		}
		idToken, err := auth.provider.Verifier(&gooidc.Config{ClientID: auth.oauth2.ClientID}).
			Verify(c.Request.Context(), rawID)
		if err != nil {
			log.Printf("Callback: ID token verification error: %v", err)
			c.String(http.StatusUnauthorized, "id_token verification failed")
			return
		}

		// Extract claims
		var claims struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := idToken.Claims(&claims); err != nil {
			log.Printf("Callback: claims extraction error: %v", err)
			c.String(http.StatusInternalServerError, "claims error")
			return
		}

		// Persist identity and raw ID token (needed for logout)
		sess.Values["user_email"] = claims.Email
		sess.Values["user_name"] = claims.Name
		sess.Values["id_token"] = rawID
		delete(sess.Values, "pkce_verifier")
		delete(sess.Values, "oauth_state")
		if err := sess.Save(c.Request, c.Writer); err != nil {
			log.Printf("Callback: session save error: %v", err)
		}

		c.Redirect(http.StatusFound, "/")
	}
}

// Logout clears the session and redirects to Okta's logout endpoint.
func Logout(auth *AuthState) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			c.Redirect(http.StatusFound, "/")
			return
		}

		sess, _ := auth.store.Get(c.Request, sessionName)
		idToken, _ := sess.Values["id_token"].(string)

		// Clear session
		sess.Options.MaxAge = -1
		sess.Save(c.Request, c.Writer)

		// Build Okta end-session URL
		var endSessionURL string
		var ep struct {
			EndSessionEndpoint string `json:"end_session_endpoint"`
		}
		if err := auth.provider.Claims(&ep); err == nil && ep.EndSessionEndpoint != "" {
			endSessionURL = ep.EndSessionEndpoint +
				"?id_token_hint=" + idToken +
				"&post_logout_redirect_uri=http://localhost:8080/"
		} else {
			endSessionURL = "/"
		}

		c.Redirect(http.StatusFound, endSessionURL)
	}
}

// AuthMiddleware reads the session on every request and sets "currentUser" in
// the Gin context so all templates can conditionally render login/logout.
func AuthMiddleware(auth *AuthState) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			c.Next()
			return
		}

		sess, err := auth.store.Get(c.Request, sessionName)
		if err == nil {
			email, _ := sess.Values["user_email"].(string)
			name, _ := sess.Values["user_name"].(string)
			if email != "" {
				c.Set("currentUser", &SessionUser{Email: email, Name: name})
			}
		}
		c.Next()
	}
}

// RequireAuth is middleware that returns 401 if there is no valid session.
func RequireAuth(auth *AuthState) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auth == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "auth not configured"})
			return
		}

		sess, err := auth.store.Get(c.Request, sessionName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session error"})
			return
		}

		email, _ := sess.Values["user_email"].(string)
		if email == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Next()
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func randomBase64URL(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func pkceChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
