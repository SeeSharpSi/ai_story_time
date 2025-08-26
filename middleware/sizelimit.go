package middleware

import (
	"fmt"
	"net/http"
)

// SizeLimitMiddleware returns a middleware function that limits request body size
func SizeLimitMiddleware(maxSize int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set the maximum request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			// Check Content-Length header if present
			if r.ContentLength > maxSize {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				errorHTML := fmt.Sprintf(`
				<div id="story-container">
					<h1>📦 Request Too Large!</h1>
					<div style="background-color: #252526; padding: 20px; border-radius: 8px; margin: 20px 0;">
						<p>🤖 Your request is too large for the story generator to process.</p>
						<p>📏 <strong>Maximum size:</strong> %d bytes</p>
						<p>💡 <strong>Suggestion:</strong> Try shortening your message or breaking it into smaller parts.</p>
						<p>🔄 <em>This helps keep the story generator running smoothly for everyone!</em></p>
					</div>
					<button onclick="window.location.reload()">Try Again</button>
				</div>`, maxSize)
				w.Write([]byte(errorHTML))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
