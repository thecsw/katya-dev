package analysis

import (
	"strings"

	"github.com/thecsw/katya/storage"
	"github.com/thecsw/katya/utils"
)

func FindRelations(texts []storage.Text, target string, width int) map[string]uint {
	fullLemas := make([]string, 0, 1000)
	for _, v := range texts {
		fullLemas = append(fullLemas, strings.Split(v.Lemmas, " ")...)
	}
	foundRelations := make(map[string]uint)
	for i, lemma := range fullLemas {
		if lemma != target {
			continue
		}
		// Start the width
		left := utils.Max(0, i-width)
		right := utils.Min(len(fullLemas), i+width)
		for j := left; j <= right; j++ {
			foundRelations[fullLemas[j]]++
		}
	}
	// delete the target word from its own relation
	delete(foundRelations, target)

	return foundRelations
}
