package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

// crawlerActionPayload is the POST body of crawler actions
type crawlerActionPayload struct {
	// Link is the crawler's link
	Link string `json:"link"`
	// OnlySubpaths tells us if we only do subdirs of the link
	OnlySubpaths bool `json:"only_subpaths"`
}

// crawlerCreator creates a crawler
func crawlerCreator(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a crawler creator payload", err, params{})
		return
	}
	user := r.Context().Value(ContextKey("user")).(User)
	thisParams := params{
		"user": user.Name,
		"link": payload.Link,
	}
	name, err := allocateCrawler(user.Name, payload.Link, payload.OnlySubpaths)
	if err != nil {
		lerr("Failed allocating a crawler in creator payload", err, thisParams)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	httpJSON(w, httpMessageReturn{"created crawler: " + name}, http.StatusOK, nil)
}

// crawlerRunner triggers a crawler
func crawlerRunner(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a crawler trigger payload", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	user := r.Context().Value(ContextKey("user")).(User)
	thisParams := params{
		"user": user.Name,
		"link": payload.Link,
	}
	name, err := triggerCrawler(user.Name, payload.Link, os.Stderr)
	if err != nil {
		lerr("Failed triggering a crawler in creator payload", err, thisParams)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	httpJSON(w, httpMessageReturn{"triggered crawler: " + name}, http.StatusOK, nil)
}

func crawlerStatusReceiver(w http.ResponseWriter, r *http.Request) {
	crawlerName := r.URL.Query().Get("name")
	if crawlerName == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("empty crawler name"))
		return
	}
	val, err := getLastScrape(crawlerName)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "getting last scrape"))
		return
	}
	httpJSON(w, *val, http.StatusOK, nil)
}
