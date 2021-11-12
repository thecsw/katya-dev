package main

import (
	"net/http"
)

// basicMiddleware simply logs every request we get
// TODO: also log the response as well
func basicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// start := time.Now()

		// uri := r.RequestURI
		// method := r.Method
		next.ServeHTTP(w, r) // serve the original request

		// duration := time.Since(start)

		// log request details
		// log.Format("request completed", log.Params{
		// 	"uri":      uri,
		// 	"method":   method,
		// 	"duration": duration,
		// })
	})
}
