package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type UserClaims struct {
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	IsAdmin       bool   `json:"isAdmin"`
}

type contextKey string

const UserKey contextKey = "user"

// localTokenPayload is the JSON payload embedded in a local auth token.
type localTokenPayload struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Exp   int64  `json:"exp"`
}

// GenerateLocalToken creates an HMAC-signed token for authenticated users.
// Format: local.<base64url(json-payload)>.<base64url(hmac-sha256)>
func GenerateLocalToken(email, name, secret string) (string, error) {
	payload := localTokenPayload{
		Email: email,
		Name:  name,
		Exp:   time.Now().Add(30 * 24 * time.Hour).Unix(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadB64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return "local." + payloadB64 + "." + sig, nil
}

// ValidateLocalToken verifies and decodes a local auth token.
func ValidateLocalToken(token, secret string) (*UserClaims, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 || parts[0] != "local" {
		return nil, fmt.Errorf("invalid token format")
	}

	payloadB64 := parts[1]
	sigB64 := parts[2]

	// Verify HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadB64))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sigB64), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, fmt.Errorf("invalid token payload")
	}

	var payload localTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("invalid token payload")
	}

	if time.Now().Unix() > payload.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &UserClaims{
		Email:         payload.Email,
		EmailVerified: "true",
		Name:          payload.Name,
	}, nil
}

// GenerateVerificationToken creates a random hex token for email verification.
func GenerateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Middleware returns an HTTP middleware that verifies the Authorization header.
// Paths starting with /api/auth/ bypass authentication.
// When devMode is true, any request is allowed through with a stub admin user identity.
func Middleware(devMode bool, adminEmails map[string]bool, jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public auth endpoints
			if strings.HasPrefix(r.URL.Path, "/api/auth/") {
				next.ServeHTTP(w, r)
				return
			}

			if devMode {
				claims := &UserClaims{
					Email:         "dev@localhost",
					EmailVerified: "true",
					Name:          "Dev User",
					Picture:       "",
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

			claims, err := ValidateLocalToken(token, jwtSecret)
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
func GetUser(ctx context.Context) *UserClaims {
	claims, _ := ctx.Value(UserKey).(*UserClaims)
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
