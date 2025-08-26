package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"story_ai/session"
)

// CSRFToken generates a random CSRF token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CSRFMiddleware provides CSRF protection
func CSRFMiddleware(manager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := manager.GetOrCreateSession(r)

			// Generate token if it doesn't exist
			if sess.CSRFToken == "" {
				token, err := generateCSRFToken()
				if err != nil {
					http.Error(w, "Failed to generate CSRF token", http.StatusInternalServerError)
					return
				}
				sess.CSRFToken = token
			}

			// For POST requests, check CSRF token
			if r.Method == http.MethodPost {
				submittedToken := r.FormValue("csrf_token")
				if submittedToken == "" {
					submittedToken = r.Header.Get("X-CSRF-Token")
				}

				if submittedToken != sess.CSRFToken {
					http.Error(w, "CSRF token mismatch", http.StatusForbidden)
					return
				}
			}

			// Add CSRF token to response headers for AJAX requests
			w.Header().Set("X-CSRF-Token", sess.CSRFToken)

			next.ServeHTTP(w, r)
		})
	}
}

// GetCSRFToken retrieves the CSRF token for a session
func GetCSRFToken(manager *session.Manager, r *http.Request) string {
	sess, _ := manager.GetOrCreateSession(r)
	if sess.CSRFToken == "" {
		token, err := generateCSRFToken()
		if err != nil {
			return ""
		}
		sess.CSRFToken = token
	}
	return sess.CSRFToken
}
