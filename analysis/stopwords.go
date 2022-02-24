package analysis

import (
	"io/ioutil"
	"strings"
)

var (
	StopwordsRU = map[string]bool{}
)

func LoadStopwords() {
	data, err := ioutil.ReadFile("analysis/stopwords/stopwords-ru.txt")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		StopwordsRU[line] = true
	}
}
