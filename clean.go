package main

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/thecsw/katya/analysis"
	"github.com/thecsw/katya/storage"
)

func cleanTexts(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}
	sourceObj, err := storage.GetSource(source, true)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	if sourceObj.Cleaned {
		httpJSON(w, "Already cleaned", http.StatusOK, nil)
		return
	}
	texts, err := storage.GetSourcesTexts(sourceObj.ID)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	newTexts, deleted, err := analysis.CleanTexts(texts)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	for _, text := range newTexts {
		fmt.Printf("[UPDATE] %s", text.URL)
		if err := storage.UpdateText(&text); err != nil {
			fmt.Println("[ERROR] FAILED", err)
		}
		fmt.Printf(" [DONE]\n")

	}
	sourceObj.Cleaned = true

	sourcesNumWordsDelta.DecrementInt(sourceObj.Link, deleted)
	globalNumWordsDelta.DecrementInt(globalDeltaCacheKey, deleted)

	err = storage.UpdateSource(sourceObj)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	httpJSON(w, "OK", http.StatusOK, nil)
}
