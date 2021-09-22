package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"net/http"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

const (
	searchResultWidth = 200
)

var (
	globalNumWordsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	globalNumSentsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	globalDeltaCacheKey = "g"

	sourcesNumWordsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	sourcesNumSentsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
)

type NoorPayload struct {
	Name         string `json:"name"`
	StartURL     string `json:"start"`
	URL          string `json:"url"`
	IP           string `json:"ip"`
	Status       int    `json:"status"`
	Text         string `json:"text"`
	Title        string `json:"title"`
	NumWords     int    `json:"num_words"`
	NumSentences int    `json:"num_sents"`
}

func noorReceiver(w http.ResponseWriter, r *http.Request) {
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
		payload.Text,
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

func updateGlobalWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(time.Minute)
		l("Starting updating the global words/sents count")
		// Update the word count
		wordDelta, _ := globalNumWordsDelta.Get(globalDeltaCacheKey)
		if err := updateGlobalWordNum(wordDelta.(uint)); err != nil {
			lerr("failed updating global word count", err, params{})
			continue
		}
		// Update the sentences count
		sentDelta, _ := globalNumSentsDelta.Get(globalDeltaCacheKey)
		if err := updateGlobalSentNum(sentDelta.(uint)); err != nil {
			lerr("failed updating global word count", err, params{})
			continue
		}
		// Drain the cache
		globalNumWordsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		globalNumSentsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		// Log the info
		l("Successfully updated the global words/sents count")
	}
}

func updateSourcesWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(time.Minute)
		l("Starting to update sources' words/sents count")
		// Update the word count
		wordItems := sourcesNumWordsDelta.Items()
		for k, v := range wordItems {
			if err := updateSourceWordNum(k, v.Object.(uint)); err != nil {
				lerr("failed updating source word count", err, params{
					"source": k,
				})
				continue
			}
			sourcesNumWordsDelta.Set(k, uint(0), cache.NoExpiration)

		}
		// Update the sents count
		sentItems := sourcesNumSentsDelta.Items()
		for k, v := range sentItems {
			if err := updateSourceSentNum(k, v.Object.(uint)); err != nil {
				lerr("failed updating source sent count", err, params{
					"source": k,
				})
				continue
			}
			sourcesNumSentsDelta.Set(k, uint(0), cache.NoExpiration)
		}
		// Log the info
		l("Successfully update sources' words/sents count")
	}
}

type StatusPayload struct {
	Name   string `json:"name"` // The name of the spider
	Status string `json:"status"`
}

func statusReceiver(w http.ResponseWriter, r *http.Request) {
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

type SearchResult struct {
	LeftReverse   string `json:"left_reverse"`
	Left          string `json:"left"`
	CenterReverse string `json:"center_reverse"`
	Center        string `json:"center"`
	Right         string `json:"right"`
	Source        string `json:"source"`
}

func textSearcher(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}

	useCSV := r.URL.Query().Get("csv")
	limitString := r.URL.Query().Get("limit")
	caseSensitive := r.URL.Query().Get("case_sensitive")

	limit, err := strconv.Atoi(limitString)
	if err != nil {
		limit = 100
	}
	resultsDB, err := findTexts(query, limit, 0, caseSensitive == "1")
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}

	results := make([]SearchResult, 0, len(resultsDB))
	for _, v := range resultsDB {
		// Try to find all indices of this substring in the text
		matches := indexStringMany(v.Text, query, caseSensitive == "1")
		for _, index := range matches {
			// Make both indices divisible by 2, so we can grab
			// 2-byte unicode values as well, without slicing
			searchWidth := searchResultWidth

			leftCrit := max(0, index-searchWidth)
			leftIndex := strings.IndexRune(v.Text[leftCrit:index], ' ')
			left := max(leftCrit, leftIndex+leftCrit)

			rightCrit := min(len(v.Text), index+len(query)+searchWidth)
			rightIndex := strings.IndexRune(v.Text[rightCrit:], ' ')
			right := max(rightCrit, rightIndex+rightCrit)

			leftText := v.Text[left+1 : index]
			centerText := v.Text[index : index+len(query)]
			rightText := v.Text[index+len(query) : right]

			toAppend := SearchResult{
				LeftReverse:   reverseString(leftText),
				Left:          leftText,
				CenterReverse: reverseString(centerText),
				Center:        centerText,
				Right:         rightText,
				Source:        v.URL,
			}

			results = append(results, toAppend)
		}
	}

	if useCSV == "1" {
		httpCSV(w, results, http.StatusOK)
		return
	}
	httpJSON(w, results, http.StatusOK, nil)
}

type crawlerActionPayload struct {
	User         string `json:"user"`
	Link         string `json:"link"`
	OnlySubpaths bool   `json:"only_subpaths"`
}

func crawlerCreator(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a crawler creator payload", err, params{})
		return
	}
	thisParams := params{
		"user": payload.User,
		"link": payload.Link,
	}
	name, err := allocateCrawler(payload.User, payload.Link, payload.OnlySubpaths)
	if err != nil {
		lerr("Failed allocating a crawler in creator payload", err, thisParams)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	httpJSON(w, httpMessageReturn{"created crawler: " + name}, http.StatusOK, nil)
}

func crawlerRunner(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a crawler trigger payload", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	thisParams := params{
		"user": payload.User,
		"link": payload.Link,
	}
	name, err := triggerCrawler(payload.User, payload.Link, os.Stderr)
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

func userCreateSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a create source payload", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = createSource(payload.User, payload.Link)
	if err != nil {
		lerr("Failed creating a source in http", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source created"}, http.StatusOK, nil)
}

func indexStringMany(s, subs string, caseSensitive bool) []int {
	if !caseSensitive {
		s = strings.ToLower(s)
		subs = strings.ToLower(subs)
	}
	res := make([]int, 0)
	start := 0
	for {
		index := strings.Index(s[start:], subs)
		if index == -1 {
			break
		}
		res = append(res, index+start)
		start = start + index + len(subs)
		if start >= len(s) {
			break
		}
	}
	return res
}
