package main

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"net/http"

	"github.com/pkg/errors"
)

const (
	searchResultWidth = 200
)

var (
	langStrToUint = map[string]uint{
		"english": 0,
		"russian": 1,
	}

	langUintToStr = map[uint]string{
		0:  "english",
		1:  "russian",
		99: "unknown",
	}
)

type NoorPayload struct {
	Name     string `json:"name"`
	StartURL string `json:"start"`
	URL      string `json:"url"`
	IP       string `json:"ip"`
	Status   int    `json:"status"`
	Text     string `json:"text"`
	Title    string `json:"title"`
	NumWords int    `json:"num_words"`
	Language string `json:"lang"`
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

	language := uint(99) // Make 99 the "default" language
	if l, found := langStrToUint[payload.Language]; found {
		language = l
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
		language,
	)
	if err != nil {
		lerr("Failed adding a new text", err, thisParams)
		httpJSON(
			w,
			nil,
			http.StatusInternalServerError,
			errors.Wrap(err, "Failed storing text in the database"),
		)
	}

	if err := updateSource(payload.StartURL, payload.NumWords); err != nil {
		lerr("failed updating source word count", err, thisParams)
		return
	}
	if err := updateGlobal(payload.NumWords); err != nil {
		lerr("failed updating global word count", err, thisParams)
		return
	}

	httpJSON(w, httpMessageReturn{
		Message: "success",
	}, http.StatusOK, nil)
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
	Language      string `json:"lang"`
}

func textSearcher(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}
	useCSV := r.URL.Query().Get("csv")
	limitString := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitString)
	if err != nil {
		limit = 100
	}
	resultsDB, err := findTexts(query, limit, 0)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}

	results := make([]SearchResult, 0, len(resultsDB))
	for _, v := range resultsDB {
		// Try to find all indices of this substring in the text
		matches := indexStringMany(v.Text, query)
		for _, index := range matches {
			// Make both indices divisible by 2, so we can grab
			// 2-byte unicode values as well, without slicing
			searchWidth := searchResultWidth
			// If it's english, then half the number of bytes
			if v.Language == 0 {
				searchWidth /= 2
			}

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
				Language:      langUintToStr[v.Language],
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
	err = createSource(payload.User, payload.Link)
	if err != nil {
		lerr("Failed creating a source in http", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source created"}, http.StatusOK, nil)
}

func indexStringMany(s, subs string) []int {
	s = strings.ToLower(s)
	subs = strings.ToLower(subs)
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
