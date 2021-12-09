package storage

import (
	"errors"
	"strings"

	"github.com/patrickmn/go-cache"
	"github.com/thecsw/katya/log"
	"gorm.io/gorm"
)

// CreateSource creates a source for a user
func CreateSource(user, link, label string) error {
	userID, err := GetUser(user, false)
	if err != nil {
		return err
	}
	toAdd := &Source{
		Link:     link,
		Label:    label,
		NumWords: 0,
	}
	source, err := GetSource(link, true)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error("Failed to check for source existence", err, log.Params{"user": user, "link": link})
		return err
	}
	// By default, set the found ID to the toAdd mention
	toAdd.ID = source.ID
	if source.ID == 0 {
		err = DB.Create(toAdd).Error
		if err != nil {
			log.Error("Failed to create a source", err, log.Params{"user": user, "link": link})
			return err
		}
	}
	err = DB.Exec("INSERT into user_sources (source_id, user_id) values (?, ?)", toAdd.ID, userID.ID).Error
	if err != nil {
		if strings.Contains(err.Error(), duplicateKeyViolatedError) {
			err = errors.New("this source is already link to the user")
		}
		log.Error("Failed to append a source", err, log.Params{"user": user, "link": link})
		return err
	}

	// Automatically enable the source
	EnableSourceByID(userID.ID, toAdd.ID)

	log.Format("Successfully created a new source", log.Params{"user": user, "link": link})
	return nil
}

// RemoveSource removes the user-link connection
func RemoveSource(user, link string) error {
	userID, err := GetUser(user, false)
	if err != nil {
		return err
	}
	source, err := GetSource(link, true)
	if err != nil {
		return err
	}
	return RemoveSourceByID(userID.ID, source.ID)
}

// RemoveSourceByID removes the user-link connection by ID
func RemoveSourceByID(user, source uint) error {
	return DB.Exec("DELETE FROM user_sources WHERE source_id = ? AND user_id = ?", source, user).Error
}

// EnableSource enables a given source for a user
func EnableSource(user, link string) error {
	userID, err := GetUser(user, false)
	if err != nil {
		return err
	}
	source, err := GetSource(link, true)
	if err != nil {
		return err
	}
	return EnableSourceByID(userID.ID, source.ID)
}

// EnableSourceByID enables a given source for a user by ID
func EnableSourceByID(user, source uint) error {
	return DB.Exec("INSERT into user_sources_enabled (source_id, user_id) values (?, ?)", source, user).Error
}

// DisableSource disables source for a user
func DisableSource(user, link string) error {
	userID, err := GetUser(user, false)
	if err != nil {
		return err
	}
	source, err := GetSource(link, true)
	if err != nil {
		return err
	}
	return DB.Exec("DELETE FROM user_sources_enabled WHERE source_id = ? AND user_id = ?", source.ID, userID.ID).Error
}

// GetSource returns the source object from database
func GetSource(source string, fill bool) (*Source, error) {
	sourceObj := &Source{}
	if ID, found := sourceToID.Get(source); found {
		// Don't ping DB to fill the object
		if !fill {
			sourceObj.ID = ID.(uint)
			return sourceObj, nil
		}
		return sourceObj, DB.First(sourceObj, ID.(uint)).Error
	}
	err := DB.Where("link = ?", source).First(sourceObj).Error
	if err != nil {
		return sourceObj, err
	}
	sourceToID.Set(source, sourceObj.ID, cache.NoExpiration)
	return sourceObj, nil
}

// IsSource checks for a source's existence
func IsSource(name string) (bool, error) {
	if _, found := sourceToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&Source{}, "link = ?", name).Count(&count).Error
	return count != 0, err
}

// UpdateSourceWordNum updates source's number of words
func UpdateSourceLabel(url, label string) error {
	return DB.Exec(
		"UPDATE sources SET label = ? WHERE link = ?",
		label, url).
		Error
}

// UpdateSourceWordNum updates source's number of words
func UpdateSourceWordNum(url string, numWords uint) error {
	return DB.Exec(
		"UPDATE sources SET num_words = num_words + ? WHERE link = ?",
		numWords, url).
		Error
}

// UpdateSourceSentNum updates source's number of sentences
func UpdateSourceSentNum(url string, numSentences uint) error {
	return DB.Exec(
		"UPDATE sources SET num_sentences = num_sentences + ? WHERE link = ?",
		numSentences, url).
		Error
}

// getSourcesTexts returns all texts of a given source
func getSourcesTexts(sourceID uint) ([]Text, error) {
	texts := make([]Text, 0, 100)
	err := DB.Model(texts).
		Joins("INNER JOIN source_texts on texts.id = source_texts.text_id").
		Joins("INNER JOIN sources on sources.id = source_texts.source_id AND sources.id = ?", sourceID).
		Find(&texts).
		Error
	return texts, err
}
