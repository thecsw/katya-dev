package main

import (
	"encoding/json"

	"net/http"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/thecsw/katya/log"
	"github.com/thecsw/katya/storage"
)

// TextPayload is what we get from our crawlers on every text submission
type TextPayload struct {
	// Name is the name of our crawler
	Name string `json:"name"`
	// StartURL is the starting URL that crawler had
	StartURL string `json:"start"`
	// URL is the URL of this text submission
	URL string `json:"url"`
	// IP is the IP address that the URL is associated with
	IP string `json:"ip"`
	// Status is the HTTP response code we received
	Status int `json:"status"`
	// Title is the title of the source webpage
	Title string `json:"title"`
	// NumWords is the number of words in this source (no punctuations)
	NumWords int `json:"num_words"`
	// NumSentences is the number of sentences in this source
	NumSentences int `json:"num_sentences"`
	// Original is the cleaned text crawler worked out
	Original string `json:"original"`
	// Text is the tokenized cleaned text SpaCy gave us
	Text string `json:"text"`
	// Shapes is the tokenized shapes data from SpaCy
	Shapes string `json:"shapes"`
	// Tags is the tokenized tags data from SpaCy
	Tags string `json:"tags"`
	// Lemmas is the tokenized lemmas data from SpaCy
	Lemmas string `json:"lemmas"`
}

// textReceiver is used by crawlers to submit a new tagged and analyzed text
func textReceiver(w http.ResponseWriter, r *http.Request) {
	scrapyLocalKey := r.Header.Get("Authorization")
	if scrapyLocalKey != "cool_local_key" {
		log.Error("Bad Authorization header", errors.New("bad key"), nil)
		return
	}
	payload := &TextPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a text payload", err, nil)
		return
	}
	thisParams := log.Params{
		"crawler": payload.Name,
		"url":     payload.URL,
		"source":  payload.StartURL,
	}
	// If it is a local upload, no need for a crawler
	if payload.Name != "LOCAL_UPLOAD" {
		// Check if such a crawler exists
		crawlerExists, err := storage.IsCrawler(payload.Name)
		if err != nil {
			log.Error("Failed checking crawler's existence", err, log.Params{
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
	}

	// Try to add the texts to the database
	err = storage.CreateText(
		payload.StartURL,
		payload.URL,
		payload.IP,
		uint(payload.Status),
		payload.Original,
		payload.Text,
		payload.Shapes,
		payload.Tags,
		payload.Lemmas,
		payload.Title,
		uint(payload.NumWords),
		uint(payload.NumSentences),
	)

	if err != nil {
		if err.Error() == "already exists" {
			// If it already exists, still link the found text to the source if not already
			httpJSON(w, httpMessageReturn{Message: "already exists"}, http.StatusOK, nil)
			return
		}
		log.Error("Failed adding a new text", err, thisParams)
		httpJSON(
			w,
			nil,
			http.StatusInternalServerError,
			errors.Wrap(err, "Failed storing text in the database"),
		)
		return
	}

	// Update the word and sent num caches
	_ = sourcesNumWordsDelta.Add(payload.StartURL, uint(0), cache.NoExpiration)
	_ = sourcesNumSentencesDelta.Add(payload.StartURL, uint(0), cache.NoExpiration)

	_, _ = sourcesNumWordsDelta.IncrementUint(payload.StartURL, uint(payload.NumWords))
	_, _ = sourcesNumSentencesDelta.IncrementUint(payload.StartURL, uint(payload.NumSentences))

	_, _ = globalNumWordsDelta.IncrementUint(globalDeltaCacheKey, uint(payload.NumWords))
	_, _ = globalNumSentencesDelta.IncrementUint(globalDeltaCacheKey, uint(payload.NumSentences))

	httpJSON(w, httpMessageReturn{
		Message: "success",
	}, http.StatusOK, nil)
}

// StatusPayload is used by crawlers to report their status
type StatusPayload struct {
	// Name is our crawler's name
	Name string `json:"name"`
	// Status is the most recent status of it
	Status string `json:"status"`
}

// statusReceiver takes the input from crawlers' statuses
func statusReceiver(w http.ResponseWriter, r *http.Request) {
	scrapyLocalKey := r.Header.Get("Authorization")
	if scrapyLocalKey != "cool_local_key" {
		log.Error("Bad Authorization header", errors.New("bad key"), nil)
		return
	}
	payload := &StatusPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a received status", err, nil)
	}

	switch payload.Status {
	case "started":
		if err := storage.CreateScrape(payload.Name); err != nil {
			log.Error("failed to log create a scrape", err, log.Params{"name": payload.Name})
		}
	case "finished":
		if err := storage.FinishScrape(payload.Name); err != nil {
			log.Error("failed to log finish a scrape", err, log.Params{"name": payload.Name})
		}
	default:
		httpJSON(w, nil, http.StatusBadRequest, errors.New("unknown status received"))
		return
	}
	httpJSON(w, httpMessageReturn{"scrape status received"}, http.StatusOK, nil)
}

// helloReceiver just sends hello through API
func helloReceiver(w http.ResponseWriter, _ *http.Request) {
	httpJSON(w, httpMessageReturn{"hello, world"}, http.StatusOK, nil)
}
