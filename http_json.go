package main

import (
	"encoding/json"
	"net/http"
)

// httpJSON is a generic http object passer.
func httpJSON(w http.ResponseWriter, data interface{}, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err != nil && status >= 400 && status < 600 {
		json.NewEncoder(w).Encode(httpErrorReturn{Error: err.Error()})
		return
	}
	json.NewEncoder(w).Encode(data)
}
