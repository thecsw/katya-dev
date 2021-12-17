package storage

import (
	"strings"

	"github.com/pkg/errors"
)

// FindTheMostFrequentWords returns a map of all standard tokens with
// the number of times they appeared within texts of a given source
func FindTheMostFrequentWords(sourceURL string) (map[string]uint, error) {
	source, err := GetSource(sourceURL, false)
	if err != nil || source.ID == 0 {
		return nil, errors.Wrap(err, "failed mapping source to ID")
	}
	texts, err := GetSourcesTexts(source.ID)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't retrieve source texts")
	}
	finalFrequencies := make(map[string]uint)
	for _, text := range texts {
		tokens := strings.Split(text.Lemmas, " ")
		for _, token := range tokens {
			lower := strings.ToLower(token)
			finalFrequencies[lower]++
		}
	}
	return finalFrequencies, err
}
