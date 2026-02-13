package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type GoogleClaims struct {
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
	IsAdmin       bool   `json:"isAdmin"`
}

type contextKey string

const UserKey contextKey = "user"

// VerifyGoogleToken validates a Google OAuth2 access token using Google's tokeninfo endpoint.
func VerifyGoogleToken(ctx context.Context, token string) (*GoogleClaims, error) {
	url := "https://oauth2.googleapis.com/tokeninfo?access_token=" + token

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("verifying token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token (status %d)", resp.StatusCode)
	}

	var claims GoogleClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("decoding claims: %w", err)
	}

	return &claims, nil
}

// Middleware returns an HTTP middleware that verifies the Authorization header
// contains a valid Google OAuth2 token. When devMode is true, any request is
// allowed through with a stub admin user identity.
// adminEmails is a set of emails that should be granted admin privileges.
func Middleware(devMode bool, adminEmails map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if devMode {
				claims := &GoogleClaims{
					Email:         "dev@localhost",
					EmailVerified: "true",
					Name:          "Dev User",
					Picture:       "",
					Sub:           "dev-local-000",
					IsAdmin:       true,
				}
				ctx := context.WithValue(r.Context(), UserKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				http.Error(w, `{"error":"invalid authorization format, use Bearer token"}`, http.StatusUnauthorized)
				return
			}

			claims, err := VerifyGoogleToken(r.Context(), token)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"unauthorized: %s"}`, err.Error()), http.StatusUnauthorized)
				return
			}

			claims.IsAdmin = adminEmails[strings.ToLower(claims.Email)]

			ctx := context.WithValue(r.Context(), UserKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser extracts the authenticated user claims from the request context.
func GetUser(ctx context.Context) *GoogleClaims {
	claims, _ := ctx.Value(UserKey).(*GoogleClaims)
	return claims
}

// RequireAdmin is an HTTP middleware that returns 403 if the user is not an admin.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user == nil || !user.IsAdmin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "admin access required"})
			return
		}
		next(w, r)
	}
}
