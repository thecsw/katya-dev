package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"unicode"
)

const (
	csvHeaderForFind        = "normal"
	csvHeaderForFrequencies = "freq"
)

var (
	// csvHeader is the CSV header when we serve a CSV file
	csvHeaders = map[string][]string{
		csvHeaderForFind: {
			"reverse left", "reverse center",
			"left", "center", "right", "source", "title"},
		csvHeaderForFrequencies: {"word", "hits"},
	}
)

// httpMessageReturn defines a generic HTTP return message.
type httpMessageReturn struct {
	Message interface{} `json:"message"`
}

// httpErrorReturn defines a generic HTTP error message.
type httpErrorReturn struct {
	Error string `json:"error"`
}

// httpCSVFindResults sends the results of SearchResult in a CSV formatted string
func httpCSVFindResults(w http.ResponseWriter, results []SearchResult, status int) {
	w.Header().Set("Content-Type", "application/csv")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	toWrite := make([][]string, 0, len(results)+1)
	toWrite = append(toWrite, csvHeaders[csvHeaderForFind])
	for _, v := range results {
		toWrite = append(toWrite, []string{
			v.LeftReverse, v.CenterReverse, v.Left, v.Center, v.Right, v.Source, v.Title,
		})
	}
	csv.NewWriter(w).WriteAll(toWrite)
}

type Pair struct {
	Key   string
	Value uint
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value > p[j].Value }

func httpCSVFreqResults(w http.ResponseWriter, results map[string]uint, status int) {
	w.Header().Set("Content-Type", "application/csv")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	toWrite := make([][]string, 0, len(results)+1)
	toWrite = append(toWrite, csvHeaders[csvHeaderForFrequencies])
	p := make(PairList, 0, len(results))
	i := 0
	for k, v := range results {
		if _, isStopword := stopwordsRU[k]; isStopword {
			continue
		}
		if unicodeIsThis(k, unicode.IsPunct) || unicodeIsThis(k, unicode.IsSymbol) {
			continue
		}
		p = append(p, Pair{k, v})
		i++
	}
	sort.Sort(p)
	for _, v := range p {
		toWrite = append(toWrite, []string{v.Key, strconv.FormatUint(uint64(v.Value), 10)})
	}
	csv.NewWriter(w).WriteAll(toWrite)
}

// httpJSON is a generic http object passer.
func httpJSON(w http.ResponseWriter, data interface{}, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err != nil && status >= 400 && status < 600 {
		json.NewEncoder(w).Encode(httpErrorReturn{Error: err.Error()})
		return
	}
	json.NewEncoder(w).Encode(data)
}

// httpHTML sends a good HTML response.
func httpHTML(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, data)
}

func unicodeIsThis(k string, isFunc func(rune) bool) bool {
	for _, r := range k {
		if !isFunc(r) {
			return false
		}
	}
	return true
}
