package analysis

import (
	"strings"

	"github.com/thecsw/katya/storage"
	"github.com/thecsw/katya/utils"
)

type Evidence struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

type Relation struct {
	Occured   int        `json:"occured"`
	Evidences []Evidence `json:"evidences"`
}

const (
	RELATION_WIDTH = 20
)

func FindRelations(texts []storage.Text, target string, width int) map[string]*Relation {
	// fullLemas := make([]string, 0, 1000)
	// for _, v := range texts {
	// 	fullLemas = append(fullLemas, strings.Split(v.Lemmas, " ")...)
	// }
	//foundRelations := make(map[string]uint)
	// for i, lemma := range fullLemas {
	// 	if lemma != target {
	// 		continue
	// 	}
	// 	// Start the width
	// 	left := utils.Max(0, i-width)
	// 	right := utils.Min(len(fullLemas), i+width)
	// 	for j := left; j <= right; j++ {
	// 		foundRelations[fullLemas[j]]++
	// 	}
	// }
	// // delete the target word from its own relation
	// delete(foundRelations, target)

	foundRelations := make(map[string]*Relation)
	for _, text := range texts {
		readable_texts := strings.Split(text.Text, " ")
		lemmas := strings.Split(text.Lemmas, " ")
		for i, lemma := range lemmas {
			if lemma != target {
				continue
			}
			// Start the width
			left := utils.Max(0, i-width)
			right := utils.Min(len(lemmas), i+width)
			wideLeft := utils.Max(0, i-RELATION_WIDTH)
			wideRight := utils.Min(len(lemmas), i+RELATION_WIDTH)

			for j := left; j <= right; j++ {
				sosed := lemmas[j]
				if _, ok := foundRelations[sosed]; !ok {
					foundRelations[sosed] = &Relation{
						Occured:   0,
						Evidences: []Evidence{},
					}
				}
				foundRelations[sosed].Occured++
				currentContext := strings.Join(readable_texts[wideLeft:wideRight], " ")
				currentContext = strings.Replace(currentContext, readable_texts[i], "?>"+readable_texts[i]+"<?", 1)
				currentContext = strings.Replace(currentContext, readable_texts[j], "!>"+readable_texts[j]+"<!", 1)
				foundRelations[sosed].Evidences = append(foundRelations[sosed].Evidences, Evidence{
					Text:   currentContext,
					Source: text.URL,
				})
			}
		}
	}
	delete(foundRelations, target)
	return foundRelations
}
