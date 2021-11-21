package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/thecsw/katya/storage"
	"github.com/thecsw/katya/utils"
)

const (
	// searchResultWidth tells us how many left-right tokens we
	// pad the center search results with
	searchResultWidth = 37
	// limitPerSource tells us how many results we will have at
	// max for each source that we find
	limitPerSource = 10
)

// SearchResult is the struct where we store the results
type SearchResult struct {
	// Left is the left context
	Left string `json:"left"`
	// LeftReverse is the reverse of the left context
	LeftReverse string `json:"left_reverse"`
	// Center is the central context
	Center string `json:"center"`
	// CenterReverse is the reverse of the central context
	CenterReverse string `json:"center_reverse"`
	// Right is the right context
	Right string `json:"right"`
	// Source is the link where the source came from
	Source string `json:"source"`
	// Title is the extract title of the source link
	Title string `json:"title"`
}

// findQueryInTexts takes /api/find query and returns a SearchResult slice
func findQueryInTexts(w http.ResponseWriter, r *http.Request) {
	// Get the actual search query, this is mission critical
	query := r.URL.Query().Get("query")
	if query == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}
	// grab the user context from the middleware
	user := r.Context().Value(ContextKey("user")).(storage.User)

	// partLookup specifies what part of the text is matched against the query
	// possible options are:
	//   - text: actual simple extracted text that's tokenized (spaces around PUNCT)
	//   - tags: tagged results, allows searching for like "NOUN PART VERB VERB"
	//   - shapes: just shapes like "Xxx xxxx - xx xxxx - x ?" -> "Это всемирную - то историю - с ?"
	//   - lemmas: lemmas will take in a nominative case of a word and search for all its
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
	if _, ok := storage.MapPartToFindFunction[partLookup]; !ok {
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

	// Find all the matches from the database by doing a string sub-match search
	resultsDB, err := storage.MapPartToFindFunction[partLookup](user.ID, query, limit, offset, caseSensitive == "1")
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
			"lemmas": v.Lemmas,
		}

		// Try to find all indices of this substring in the text to later map it to token indices
		matches := utils.StringsIndexMultiple(whatToSearchIn[partLookup], query, caseSensitive == "1")

		// If there are no matches found (DB lied???) then we skip this
		if len(matches) < 1 {
			continue
		}

		// Split the text sections into the actual token slice
		textSplit := strings.Split(v.Text, " ")
		tagsSplit := strings.Split(v.Tags, " ")
		shapesSplit := strings.Split(v.Shapes, " ")
		lemmasSplit := strings.Split(v.Lemmas, " ")

		// File every match in the found text in its own result case
		for _, index := range matches[:utils.Min(limitPerSource, len(matches))] {
			// If we hit a bad index, skip and continue
			if index < 1 {
				continue
			}

			// this maps what token split we will be using for mapping the index to token index
			whereToFindTheTokenIndex := map[string][]string{
				"text":   textSplit,
				"shapes": shapesSplit,
				"tags":   tagsSplit,
				"lemmas": lemmasSplit,
			}

			// Map the actual found query's index into the token index
			resultsSplitLeftIndex := utils.FindTokenIndex(whereToFindTheTokenIndex[partLookup], index)
			resultsSplitRightIndex := utils.FindTokenIndex(whereToFindTheTokenIndex[partLookup], index+len(query)) + 1

			// Find the indices that we will split the tokens from left to right
			leftSplitLeftIndex := utils.Max(0, resultsSplitLeftIndex-searchResultWidth)
			leftSplitRightIndex := resultsSplitLeftIndex
			centerSplitLeftIndex := resultsSplitLeftIndex
			centerSplitRightIndex := resultsSplitRightIndex
			rightSplitLeftIndex := resultsSplitRightIndex
			rightSplitRightIndex := utils.Min(len(textSplit), resultsSplitRightIndex+searchResultWidth)

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
			// leftNominativesSplit := lemmasSplit[leftSplitLeftIndex:leftSplitRightIndex]
			// centerNominativesSplit := lemmasSplit[centerSplitLeftIndex:centerSplitRightIndex]
			// rightNominativesSplit := lemmasSplit[rightSplitLeftIndex:rightSplitRightIndex]

			// Join the tokens into the actual representable state for the user
			leftText := strings.Join(leftTextSplit, " ")
			centerText := strings.Join(centerTextSplit, " ")
			rightText := strings.Join(rightTextSplit, " ")

			// Create the object that we will be serving
			toAppend := SearchResult{
				LeftReverse:   utils.ReverseString(leftText),
				Left:          leftText,
				CenterReverse: utils.ReverseString(centerText),
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
		httpCSVFindResults(w, results, http.StatusOK)
		return
	}

	// Fallback to the default JSON return
	httpJSON(w, results, http.StatusOK, nil)
}

// frequencyFinder returns a word frequency table for a given source
func frequencyFinder(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("bad query"))
		return
	}
	result, err := storage.FindTheMostFrequentWords(source)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "oops"))
		return
	}
	httpCSVFreqResults(w, result, http.StatusOK)
}
