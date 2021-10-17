package main

import (
	"encoding/json"

	"net/http"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

type NoorPayload struct {
	Name         string `json:"name"`
	StartURL     string `json:"start"`
	URL          string `json:"url"`
	IP           string `json:"ip"`
	Status       int    `json:"status"`
	Original     string `json:"original"`
	Text         string `json:"text"`
	Shapes       string `json:"shapes"`
	Tags         string `json:"tags"`
	Nominatives  string `json:"nomins"`
	Title        string `json:"title"`
	NumWords     int    `json:"num_words"`
	NumSentences int    `json:"num_sents"`
}

func noorReceiver(w http.ResponseWriter, r *http.Request) {
	noorKey := r.Header.Get("Authorization")
	if noorKey != "noorkey" {
		lerr("Bad Authorization header", errors.New("bad key"), params{})
		return
	}
	payload := &NoorPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a noor payload", err, params{})
		return
	}
	thisParams := params{
		"crawler": payload.Name,
		"url":     payload.URL,
		"source":  payload.StartURL,
	}
	// Check if such a crawler exists
	crawlerExists, err := isCrawler(payload.Name)
	if err != nil {
		lerr("Failed checking crawler's existence", err, params{
			"crawler": payload.Name,
		})
	}
	if !crawlerExists {
		httpJSON(
			w,
			nil,
			http.StatusForbidden,
			errors.New("this crawler doesn't exist"),
		)
		return
	}

	// Try to add the texts to the database
	err = createText(
		payload.StartURL,
		payload.URL,
		payload.IP,
		uint(payload.Status),
		payload.Original,
		payload.Text,
		payload.Shapes,
		payload.Tags,
		payload.Nominatives,
		payload.Title,
		uint(payload.NumWords),
		uint(payload.NumSentences),
	)

	if err != nil {
		if err.Error() == "already exists" {
			httpJSON(w, httpMessageReturn{Message: "already exists"}, http.StatusOK, nil)
			return
		}
		lerr("Failed adding a new text", err, thisParams)
		httpJSON(
			w,
			nil,
			http.StatusInternalServerError,
			errors.Wrap(err, "Failed storing text in the database"),
		)
		return
	}

	// Update the word and sent num caches
	sourcesNumWordsDelta.Add(payload.StartURL, uint(0), cache.NoExpiration)
	sourcesNumSentsDelta.Add(payload.StartURL, uint(0), cache.NoExpiration)

	sourcesNumWordsDelta.IncrementUint(payload.StartURL, uint(payload.NumWords))
	sourcesNumSentsDelta.IncrementUint(payload.StartURL, uint(payload.NumSentences))

	globalNumWordsDelta.IncrementUint(globalDeltaCacheKey, uint(payload.NumWords))
	globalNumSentsDelta.IncrementUint(globalDeltaCacheKey, uint(payload.NumSentences))

	httpJSON(w, httpMessageReturn{
		Message: "success",
	}, http.StatusOK, nil)
}

type StatusPayload struct {
	Name   string `json:"name"` // The name of the spider
	Status string `json:"status"`
}

func statusReceiver(w http.ResponseWriter, r *http.Request) {
	noorKey := r.Header.Get("Authorization")
	if noorKey != "noorkey" {
		lerr("Bad Authorization header", errors.New("bad key"), params{})
		return
	}
	payload := &StatusPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a received status", err, params{})
	}

	switch payload.Status {
	case "started":
		createScrape(payload.Name)
	case "finished":
		finishScrape(payload.Name)
	default:
		httpJSON(w, nil, http.StatusBadRequest, errors.New("unknown status received"))
		return
	}
	httpJSON(w, httpMessageReturn{"scrape status received"}, http.StatusOK, nil)
}

func findTokenIndex(tokens []string, index int) int {
	currentSum := 0
	for i, v := range tokens {
		if currentSum > index {
			return i - 1
		}
		currentSum += len(v) + 1
	}
	return -1
}

func userGetSources(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(ContextKey("user")).(User)
	sources, err := getUserSources(user.Name)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "failed to retrieve sources"))
		return
	}
	httpJSON(w, sources, http.StatusOK, nil)
}

func userCreateSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a create source payload", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	user := r.Context().Value(ContextKey("user")).(User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = createSource(user.Name, payload.Link)
	if err != nil {
		lerr("Failed creating a source in http", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source created"}, http.StatusOK, nil)
}

func userDeleteSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a create source payload", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, errors.Wrap(err, "bad request payload"))
		return
	}
	user := r.Context().Value(ContextKey("user")).(User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = removeSource(user.Name, payload.Link)
	if err != nil {
		lerr("Failed deleting a user source in http", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source deleted"}, http.StatusOK, nil)
}

func helloReceiver(w http.ResponseWriter, r *http.Request) {
	httpJSON(w, httpMessageReturn{"hello, world"}, http.StatusOK, nil)
}
