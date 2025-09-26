package web

import (
	"log"
	"net/http"
	"runtime/debug"
)
//quite a useless middleware ngl (err handling final boss lol)

func panicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic occurred: %v\n%s", err, debug.Stack())
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
