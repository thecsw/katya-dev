package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"net/http"

	"github.com/pkg/errors"
)

const (
	searchResultWidth = 200
)

type NoorPayload struct {
	Name     string `json:"name"`
	StartURL string `json:"start"`
	URL      string `json:"url"`
	IP       string `json:"ip"`
	Status   int    `json:"status"`
	Text     string `json:"text"`
	NumWords int    `json:"num_words"`
}

func noorReceiver(w http.ResponseWriter, r *http.Request) {
	payload := &NoorPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a noor payload", err, params{})
		return
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
		payload.Text, uint(payload.NumWords),
	)
	if err != nil {
		lerr("Failed adding a new text", err, params{
			"crawler": payload.Name,
			"url":     payload.URL,
			"source":  payload.StartURL,
		})
		httpJSON(
			w,
			nil,
			http.StatusInternalServerError,
			errors.Wrap(err, "Failed storing text in the database"),
		)
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

	w.WriteHeader(http.StatusOK)
}

type SearchResult struct {
	Left   string `json:"left"`
	Center string `json:"center"`
	Right  string `json:"right"`
	Source string `json:"source"`
	URL    string `json:"url"`
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
			leftCrit := max(0, index-searchResultWidth)
			leftIndex := strings.IndexRune(v.Text[leftCrit:index], ' ')
			left := max(leftCrit, leftIndex+leftCrit)

			rightCrit := min(len(v.Text), index+len(query)+searchResultWidth)
			rightIndex := strings.IndexRune(v.Text[rightCrit:], ' ')
			right := max(rightCrit, rightIndex+rightCrit)

			toAppend := SearchResult{
				Left:   v.Text[left+1 : index],
				Center: v.Text[index : index+len(query)],
				Right:  v.Text[index+len(query) : right],
				Source: v.URL,
				URL:    v.URL,
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
