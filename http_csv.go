package main

import (
	"encoding/csv"
	"net/http"
)

var (
	// csvHeader is the CSV header when we serve a CSV file
	csvHeader = []string{
		"reverse left", "reverse center", "left", "center", "right", "source", "title",
	}
)

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
