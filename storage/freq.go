package storage

import (
	"strings"

	"github.com/pkg/errors"
)

// FindTheMostFrequentWords returns a map of all standard tokens with
// the number of times they appeared within texts of a given source
func FindTheMostFrequentWords(sourceID uint) (map[string]uint, error) {
	texts, err := getSourcesTexts(sourceID)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve source texts")
	}
	finalFrequencies := make(map[string]uint)
	for _, text := range texts {
		tokens := strings.Split(text.Nominatives, " ")
		for _, token := range tokens {
			lower := strings.ToLower(token)
			finalFrequencies[lower]++
		}
	}
	return finalFrequencies, err
}
