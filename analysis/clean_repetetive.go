package analysis

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/thecsw/katya/storage"
)

func CleanTexts(texts []storage.Text) ([]storage.Text, int, error) {
	if len(texts) <= 2 {
		return nil, 0, errors.New("not enough texts provided, need at least 2")
	}

	fmt.Println("Starting cleanup!")
	totalDeleted := 0
	// extract the original tokens
	firstText := strings.Split(texts[0].Text, " ")
	stt := LCSIM(firstText, strings.Split(texts[1].Text, " "))

	for i := 1; i < len(texts); i++ {
		fmt.Printf("[CLEAN] [%d/%d] Processing %s", i, len(texts)-1, texts[i].URL)
		text := strings.Split(texts[i].Text, " ")
		shapes := strings.Split(texts[i].Shapes, " ")
		tags := strings.Split(texts[i].Tags, " ")
		lemmas := strings.Split(texts[i].Lemmas, " ")

		st := LCSIM(firstText, text)

		textNew, deleted := removeIndices(text, st.Right)
		shapesNew, _ := removeIndices(shapes, st.Right)
		tagsNew, _ := removeIndices(tags, st.Right)
		lemmasNew, _ := removeIndices(lemmas, st.Right)

		texts[i].Text = strings.Join(textNew, " ")
		texts[i].Shapes = strings.Join(shapesNew, " ")
		texts[i].Tags = strings.Join(tagsNew, " ")
		texts[i].Lemmas = strings.Join(lemmasNew, " ")
		texts[i].NumWords -= uint(deleted)

		totalDeleted += deleted

		fmt.Printf(" [DELETED %d]", deleted)

		fmt.Printf(" Done!\n")
	}
	// Update the final pivot text
	fmt.Println("Finally updating the pivot")
	textNew, deleted := removeIndices(strings.Split(texts[0].Text, " "), stt.Right)
	shapesNew, _ := removeIndices(strings.Split(texts[0].Shapes, " "), stt.Right)
	tagsNew, _ := removeIndices(strings.Split(texts[0].Tags, " "), stt.Right)
	lemmasNew, _ := removeIndices(strings.Split(texts[0].Lemmas, " "), stt.Right)

	texts[0].Text = strings.Join(textNew, " ")
	texts[0].Shapes = strings.Join(shapesNew, " ")
	texts[0].Tags = strings.Join(tagsNew, " ")
	texts[0].Lemmas = strings.Join(lemmasNew, " ")
	texts[0].NumWords -= uint(deleted)

	return texts, totalDeleted, nil
}

func removeIndices(texts []string, indices []int) ([]string, int) {
	indicesPointer := 0
	actuallyRemoved := 0
	toReturn := make([]string, 0, len(texts)-len(indices))
	for i, v := range texts {
		// Sanity check
		if len(v) == 0 {
			continue
		}
		// see if it's a bad symbol
		isTooCommon := unicodeIsThis(v, unicode.IsPunct)
		// Check that it's not a stopword or a punct
		if i == indices[indicesPointer] {
			indicesPointer++
			// If it's a symbol that's too common (period), add it
			if !isTooCommon {
				actuallyRemoved++
				continue
			}
		}
		toReturn = append(toReturn, v)
	}
	return toReturn, actuallyRemoved
}

func getIndices(texts []string, indices []int) []string {
	toReturn := make([]string, len(indices))
	for _, v := range indices {
		toReturn = append(toReturn, texts[v])
	}
	return toReturn
}
