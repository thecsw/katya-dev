package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	// csvHeader is the CSV header when we serve a CSV file
	csvHeader = []string{
		"reverse left", "reverse center", "left", "center", "right", "source", "title",
	}
)

// httpMessageReturn defines a generic HTTP return message.
type httpMessageReturn struct {
	Message interface{} `json:"message"`
}

// httpErrorReturn defines a generic HTTP error message.
type httpErrorReturn struct {
	Error string `json:"error"`
}

// httpCSV sends the results of SearchResult in a CSV formatted string
func httpCSV(w http.ResponseWriter, results []SearchResult, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	toWrite := make([][]string, 0, len(results)+1)
	toWrite = append(toWrite, csvHeader)
	for _, v := range results {
		toWrite = append(toWrite, []string{
			v.LeftReverse, v.CenterReverse, v.Left, v.Center, v.Right, v.Source, v.Title,
		})
	}
	csv.NewWriter(w).WriteAll(toWrite)
}

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

// httpHTML sends a good HTML response.
func httpHTML(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, data)
}
