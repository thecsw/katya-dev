package storage

import (
	"strings"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/thecsw/katya/log"
	"gorm.io/gorm"
)

const (
	duplicateKeyViolatedError = "duplicate key value violates unique constraint"
)

// CreateText creates a full text that we receive from our scrapers
func CreateText(
	source string,
	url string,
	ip string,
	status uint,
	original string,
	text string,
	shapes string,
	tags string,
	lemmas string,
	title string,
	numWords uint,
	numSentences uint,
) error {
	textFound, err := GetText(url, false)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error("failed the text existence check", err, log.Params{"url": url})
		return errors.Wrap(err, "failed the text existence check")
	}
	// If there is no text existing with the URL, then make one
	var toAdd *Text
	alreadyExisted := textFound.ID != 0
	if !alreadyExisted {
		toAdd = &Text{
			URL:          url,
			IP:           ip,
			Status:       status,
			Original:     original,
			Text:         text,
			Shapes:       shapes,
			Tags:         tags,
			Lemmas:       lemmas,
			Title:        title,
			NumWords:     numWords,
			NumSentences: numSentences,
			Sources:      []*Source{},
		}
		err = DB.Create(toAdd).Error
		if err != nil {
			log.Error("failed to create a new text", err, log.Params{"url": url})
			return errors.Wrap(err, "failed to create a new text")
		}
		textFound.ID = toAdd.ID
	}
	// Try to link the source to the text, which already exist or was just created
	sourceObj, err := GetSource(source, false)
	if err != nil {
		log.Error("failed source retrieval in text creation", err, log.Params{"source": source})
		return errors.Wrap(err, "failed source retrieval in text creation")
	}
	// Connect the source to the text
	err = TextConnectSource(sourceObj.ID, textFound.ID)
	alreadyLinkedToSource := false
	if err != nil {
		if strings.Contains(err.Error(), duplicateKeyViolatedError) {
			alreadyLinkedToSource = true
		} else {
			log.Error("could not link text to source", err, log.Params{})
			return err
		}

	}
	log.Format("Successfully created a new text", log.Params{
		"url":             url,
		"title":           title,
		"ip":              ip,
		"num_words":       numWords,
		"num_sentences":   numSentences,
		"already_existed": alreadyExisted,
		"already_linked":  alreadyLinkedToSource,
	})
	return nil
}

// IsText checks whether the url already exists
func IsText(url string) (bool, error) {
	if _, found := urlToID.Get(url); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&User{}, "url = ?", url).Count(&count).Error
	return count != 0, err
}

// GetText returns a text by the url
func GetText(url string, fill bool) (*Text, error) {
	text := &Text{}
	if ID, found := urlToID.Get(url); found {
		// Don't ping DB to fill the object
		if !fill {
			text.ID = ID.(uint)
			return text, nil
		}
		return text, DB.First(text, ID.(uint)).Error
	}
	err := DB.First(text, "url = ?", url).Error
	if err != nil {
		return text, err
	}
	urlToID.Set(url, text.ID, cache.NoExpiration)
	return text, nil
}

// TextConnectSource will add a manual relationship between a source and a text
func TextConnectSource(sourceID, textID uint) error {
	return DB.Exec("INSERT into source_texts (source_id, text_id) values (?, ?)", sourceID, textID).Error
}

// findTexts is a general matcher that takes a username and runs it
// func findTexts(
// 	user string,
// 	query string,
// 	limit int,
// 	offset int,
// 	caseSensitive bool,
// ) ([]Text, error) {
// 	texts := make([]Text, 0, limit)
// 	sqlWhere := "texts.text LIKE ?"
// 	sqlMatch := "%" + query + "%"
// 	if !caseSensitive {
// 		sqlWhere = "lower(texts.text) LIKE ?"
// 		sqlMatch = "%" + strings.ToLower(query) + "%"
// 	}
// 	err := DB.Model(texts).
// 		Joins("JOIN source_texts on texts.id = source_texts.text_id").
// 		Joins("JOIN sources on sources.id = source_texts.source_id").
// 		Joins("JOIN user_sources on sources.id = user_sources.source_id").
// 		Joins("JOIN users on user_sources.user_id = users.id AND users.name = ?", user).
// 		Where(sqlWhere, sqlMatch).
// 		Limit(limit).
// 		Offset(offset).
// 		Find(&texts).
// 		Error
// 	return texts, err
// }
