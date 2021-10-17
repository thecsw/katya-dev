package main

import (
	"fmt"
	"net/http"
)

// httpHTML sends a good HTML response.
func httpHTML(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, data)
}
