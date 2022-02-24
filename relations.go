package main

import (
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/thecsw/katya/analysis"
	"github.com/thecsw/katya/storage"
)

func findRelations(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}
	target := r.URL.Query().Get("target")
	if target == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad target"))
		return
	}
	widthT := r.URL.Query().Get("width")
	if widthT == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad width"))
		return
	}
	width, _ := strconv.Atoi(widthT)
	sourceObj, err := storage.GetSource(source, true)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	texts, err := storage.GetSourcesTexts(sourceObj.ID)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	relations := analysis.FindRelations(texts, target, width)

	sorted := analysis.FilterStopwords(relations, analysis.StopwordsRU)

	httpJSON(w, sorted, http.StatusOK, nil)
}
