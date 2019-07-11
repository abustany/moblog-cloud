package middlewares

import (
	"log"
	"net/http"
)

func WithLogging(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling %s request to %s (headers: %v)", r.Method, r.URL.String(), r.Header)

		handler.ServeHTTP(w, r)
	})
}
