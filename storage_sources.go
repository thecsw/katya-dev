package main

import (
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// createSource creates a source for a user
func createSource(user, link string) error {
	userID, err := getUser(user, false)
	if err != nil {
		return err
	}
	toAdd := &Source{
		Link:     link,
		NumWords: 0,
	}
	source, err := getSource(link, true)
	if err != nil && err != gorm.ErrRecordNotFound {
		lerr("Failed to check for source existince", err, params{"user": user, "link": link})
		return err
	}
	if source.ID == 0 {
		err = DB.Create(toAdd).Error
		if err != nil {
			lerr("Failed to create a source", err, params{"user": user, "link": link})
			return err
		}
	}
	err = DB.Exec("INSERT into user_sources (source_id, user_id) values (?, ?)", toAdd.ID, userID.ID).Error
	if err != nil {
		lerr("Failed to append a source", err, params{"user": user, "link": link})
		return err
	}
	lf("Successfully created a new source", params{"user": user, "link": link})
	return nil
}

// removeSource removes the user-link connection
func removeSource(user, link string) error {
	userID, err := getUser(user, false)
	if err != nil {
		return err
	}
	source, err := getSource(link, true)
	if err != nil {
		return err
	}
	return DB.Exec("DELETE FROM user_sources WHERE source_id = ? AND user_id = ?", source.ID, userID.ID).Error
}

// getSource returns the source object from database
func getSource(source string, fill bool) (*Source, error) {
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

// isSource checks for a source's existence
func isSource(name string) (bool, error) {
	if _, found := sourceToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&Source{}, "link = ?", name).Count(&count).Error
	return count != 0, err
}

// updateSourceWordNum updates source's number of words
func updateSourceWordNum(url string, numWords uint) error {
	return DB.Exec(
		"UPDATE sources SET num_words = num_words + ? WHERE link = ?",
		numWords, url).
		Error
}

// updateSourceSentNum updates source's number of sentences
func updateSourceSentNum(url string, numSents uint) error {
	return DB.Exec(
		"UPDATE sources SET num_sentences = num_sentences + ? WHERE link = ?",
		numSents, url).
		Error
}
