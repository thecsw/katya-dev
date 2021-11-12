package main

import (
	"io/ioutil"
	"strings"
)

var (
	stopwordsRU = map[string]bool{}
)

func loadStopwords() {
	data, err := ioutil.ReadFile("data/stopwords/stopwords-ru.txt")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		stopwordsRU[line] = true
	}
}
