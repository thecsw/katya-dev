package main

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/thecsw/katya/analysis"
)

// frequencyFinder returns a word frequency table for a given source
func frequencyFinder(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}
	// whether we should serve a CSV file instead of a JSON
	useCSV := r.URL.Query().Get("csv")
	result, err := analysis.FindTheMostFrequentWords(source)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	if useCSV == "1" {
		httpCSVFreqResults(w, result, http.StatusOK)
		return
	}
	httpJSON(w, analysis.FilterStopwordsSimple(result, analysis.StopwordsRU), http.StatusOK, nil)
}
