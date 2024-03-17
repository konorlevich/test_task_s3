package middleware

import (
	"net/http"
)

// CheckAuth we should be an ACL, but this is good for now
func CheckAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		username, _, ok := r.BasicAuth()
		if !ok || username == "" {
			rw.WriteHeader(http.StatusUnauthorized)
			_, _ = rw.Write([]byte("you are not authorized for this action"))
			return
		}
		next.ServeHTTP(rw, r)
	})
}
