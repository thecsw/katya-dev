package storage

import (
	"github.com/patrickmn/go-cache"
	"github.com/thecsw/katya/log"
	"gorm.io/gorm"
)

// CreateSource creates a source for a user
func CreateSource(user, link string) error {
	userID, err := GetUser(user, false)
	if err != nil {
		return err
	}
	toAdd := &Source{
		Link:     link,
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
		log.Error("Failed to append a source", err, log.Params{"user": user, "link": link})
		return err
	}
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
	return DB.Exec("DELETE FROM user_sources WHERE source_id = ? AND user_id = ?", source.ID, userID.ID).Error
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
func UpdateSourceWordNum(url string, numWords uint) error {
	return DB.Exec(
		"UPDATE sources SET num_words = num_words + ? WHERE link = ?",
		numWords, url).
		Error
}

// UpdateSourceSentNum updates source's number of sentences
func UpdateSourceSentNum(url string, numSents uint) error {
	return DB.Exec(
		"UPDATE sources SET num_sentences = num_sentences + ? WHERE link = ?",
		numSents, url).
		Error
}