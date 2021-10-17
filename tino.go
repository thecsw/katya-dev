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
	searchResultWidth = 37
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

func updateGlobalWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(time.Minute)
		actuallyUpdated := false
		//l("Starting updating the global words/sents count")
		// Update the word count
		wordDelta, _ := globalNumWordsDelta.Get(globalDeltaCacheKey)
		if wordDelta.(uint) != 0 {
			if err := updateGlobalWordNum(wordDelta.(uint)); err != nil {
				lerr("failed updating global word count", err, params{})
				continue
			}
			actuallyUpdated = true
		}
		// Update the sentences count
		sentDelta, _ := globalNumSentsDelta.Get(globalDeltaCacheKey)
		if sentDelta.(uint) != 0 {
			if err := updateGlobalSentNum(sentDelta.(uint)); err != nil {
				lerr("failed updating global word count", err, params{})
				continue
			}
			actuallyUpdated = true
		}
		// Drain the cache
		globalNumWordsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		globalNumSentsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		// Log the info
		if actuallyUpdated {
			l("Successfully updated the global words/sents count")
		}
	}
}

func updateSourcesWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(time.Minute)
		actuallyUpdated := false
		//l("Starting to update sources' words/sents count")
		// Update the word count
		wordItems := sourcesNumWordsDelta.Items()
		for k, v := range wordItems {
			delta := v.Object.(uint)
			if delta == 0 {
				continue
			}
			if err := updateSourceWordNum(k, delta); err != nil {
				lerr("failed updating source word count", err, params{
					"source": k,
				})
				continue
			}
			actuallyUpdated = true
			sourcesNumWordsDelta.Set(k, uint(0), cache.NoExpiration)

		}
		// Update the sents count
		sentItems := sourcesNumSentsDelta.Items()
		for k, v := range sentItems {
			delta := v.Object.(uint)
			if delta == 0 {
				continue
			}
			if err := updateSourceSentNum(k, delta); err != nil {
				lerr("failed updating source sent count", err, params{
					"source": k,
				})
				continue
			}
			actuallyUpdated = true
			sourcesNumSentsDelta.Set(k, uint(0), cache.NoExpiration)
		}
		// Log the info
		if actuallyUpdated {
			l("Successfully update sources' words/sents count")
		}
	}
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

type SearchResult struct {
	LeftReverse   string `json:"left_reverse"`
	Left          string `json:"left"`
	CenterReverse string `json:"center_reverse"`
	Center        string `json:"center"`
	Right         string `json:"right"`
	Source        string `json:"source"`
	Title         string `json:"title"`
}

func textSearcher(w http.ResponseWriter, r *http.Request) {
	// Get the actual search query, this is mission critical
	query := r.URL.Query().Get("query")
	if query == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}
	// grab the user context from the middleware
	user := r.Context().Value(ContextKey("user")).(User)

	// partLookup specifies what part of the text is matched against the query
	// possible options are:
	//   - text: actual simple extracted text that's tokenized (spaces around PUNCT)
	//   - tags: tagged results, allows searching for like "NOUN PART VERB VERB"
	//   - shapes: just shapes like "Xxx xxxx - xx xxxx - x ?" -> "Это всемирную - то историю - с ?"
	//   - nomins: nominatives will take in a nominative case of a word and search for all its
	//             conjugations, such that a search for a nominative word of "полюбить" will
	//             automatically search for "полюбил" or "полюбить" or "полюбили". Pretty coll!
	partLookup := r.URL.Query().Get("part")
	// whether we should serve a CSV file instead of a JSON
	useCSV := r.URL.Query().Get("csv")
	// how many results do we want to show
	limitString := r.URL.Query().Get("limit")
	// the offset to pass to the DB for results
	offsetString := r.URL.Query().Get("offset")
	// whether we should care for casing in DB string match
	caseSensitive := r.URL.Query().Get("case_sensitive")

	// Fallback to a by-text lookup if not given or bad
	if _, ok := findByPart[partLookup]; !ok {
		partLookup = "text"
	}

	// Convert limit to int, fallback to 100
	limit, err := strconv.Atoi(limitString)
	if err != nil || limit > 100 || limit < 0 {
		limit = 100
	}

	// Convert offset to int, fallback to 0
	offset, err := strconv.Atoi(offsetString)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Find all the matches from the database by doing a string submatch search
	resultsDB, err := findByPart[partLookup](user.ID, query, limit, offset, caseSensitive == "1")
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}

	// Create the final object we will be serving through the API
	results := make([]SearchResult, 0, len(resultsDB))
	for _, v := range resultsDB {

		// This map allows us to dynamically choose the text part that we used for DB string search
		whatToSearchIn := map[string]string{
			"text":   v.Text,
			"shapes": v.Shapes,
			"tags":   v.Tags,
			"nomins": v.Nominatives,
		}

		// Try to find all indices of this substring in the text to later map it to token indices
		matches := indexStringMany(whatToSearchIn[partLookup], query, caseSensitive == "1")

		// If there are no matches found (DB lied???) then we skip this
		if len(matches) < 1 {
			continue
		}

		// Split the text sections into the actual token slice
		textSplit := strings.Split(v.Text, " ")
		tagsSplit := strings.Split(v.Tags, " ")
		shapesSplit := strings.Split(v.Shapes, " ")
		nominativesSplit := strings.Split(v.Nominatives, " ")

		// File every match in the found text in its own result case
		for _, index := range matches {
			// If we hit a bad index, skip and continue
			if index < 1 {
				continue
			}

			// this maps what token split we will be using for mapping the index to token index
			whereToFindTheTokenIndex := map[string][]string{
				"text":   textSplit,
				"shapes": shapesSplit,
				"tags":   tagsSplit,
				"nomins": nominativesSplit,
			}

			// Map the actual found query's index into the token index
			resultsSplitLeftIndex := findTokenIndex(whereToFindTheTokenIndex[partLookup], index)
			resultsSplitRightIndex := findTokenIndex(whereToFindTheTokenIndex[partLookup], index+len(query)) + 1

			// Find the indices that we will split the tokens from left to right
			leftSplitLeftIndex := max(0, resultsSplitLeftIndex-searchResultWidth)
			leftSplitRightIndex := resultsSplitLeftIndex
			centerSplitLeftIndex := resultsSplitLeftIndex
			centerSplitRightIndex := resultsSplitRightIndex
			rightSplitLeftIndex := resultsSplitRightIndex
			rightSplitRightIndex := min(len(textSplit), resultsSplitRightIndex+searchResultWidth)

			// Split the text tokens into the results section
			leftTextSplit := textSplit[leftSplitLeftIndex:leftSplitRightIndex]
			centerTextSplit := textSplit[centerSplitLeftIndex:centerSplitRightIndex]
			rightTextSplit := textSplit[rightSplitLeftIndex:rightSplitRightIndex]

			// // Split the tags tokens into the results section
			// leftTagsSplit := tagsSplit[leftSplitLeftIndex:leftSplitRightIndex]
			// centerTagsSplit := tagsSplit[centerSplitLeftIndex:centerSplitRightIndex]
			// rightTagsSplit := tagsSplit[rightSplitLeftIndex:rightSplitRightIndex]

			// // Split the shapes tokens into the results section
			// leftShapesSplit := shapesSplit[leftSplitLeftIndex:leftSplitRightIndex]
			// centerShapesSplit := shapesSplit[centerSplitLeftIndex:centerSplitRightIndex]
			// rightShapesSplit := shapesSplit[rightSplitLeftIndex:rightSplitRightIndex]

			// // Split the nominative tokens into the results section
			// leftNominativesSplit := nominativesSplit[leftSplitLeftIndex:leftSplitRightIndex]
			// centerNominativesSplit := nominativesSplit[centerSplitLeftIndex:centerSplitRightIndex]
			// rightNominativesSplit := nominativesSplit[rightSplitLeftIndex:rightSplitRightIndex]

			// Join the tokens into the actual representable state for the user
			leftText := strings.Join(leftTextSplit, " ")
			centerText := strings.Join(centerTextSplit, " ")
			rightText := strings.Join(rightTextSplit, " ")

			// Create the object that we will be serving
			toAppend := SearchResult{
				LeftReverse:   reverseString(leftText),
				Left:          leftText,
				CenterReverse: reverseString(centerText),
				Center:        centerText,
				Right:         rightText,
				Source:        v.URL,
				Title:         v.Title,
			}

			// Append it to the final results
			results = append(results, toAppend)
		}
	}

	// Override the serving into the CSV serving function
	if useCSV == "1" {
		httpCSV(w, results, http.StatusOK)
		return
	}

	// Fallback to the default JSON return
	httpJSON(w, results, http.StatusOK, nil)
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

type crawlerActionPayload struct {
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
