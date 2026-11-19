package httptransport

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

type csrfContextKey struct{}

const csrfCookie = "cup_csrf"
const csrfHeader = "X-CSRF-Token"

func randomToken() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func csrfMiddleware(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(csrfCookie)
			token := ""
			if err == nil {
				token = cookie.Value
			}
			if token == "" {
				token = randomToken()
				http.SetCookie(w, &http.Cookie{
					Name: csrfCookie, Value: token, Path: "/", HttpOnly: false,
					Secure: secure, SameSite: http.SameSiteLaxMode,
				})
			}
			if unsafeMethod(r.Method) && !csrfExempt(r.URL.Path) {
				got := r.Header.Get(csrfHeader)
				if cookie == nil || subtle.ConstantTimeCompare([]byte(got), []byte(cookie.Value)) != 1 {
					http.Error(w, "csrf", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), csrfContextKey{}, token)))
		})
	}
}

func unsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func csrfExempt(path string) bool {
	return strings.HasPrefix(path, "/api/v1/auth/s95/")
}
