package main

import (
	"errors"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// createUser creates a user in the database
func createUser(name, pass string) error {
	found, err := isUser(name)
	if found {
		lerr("User already exists", err, params{"user": name})
		return errors.New("User already exists")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		lerr("Failed to check user existence", err, params{"user": name})
		return err
	}
	err = DB.Create(&User{Name: name, Password: shaEncode(pass)}).Error
	if err != nil {
		return err
	}
	lf("Successfully created a new user", params{"name": name})
	return nil
}

// getUser gets a user from the database by the username
func getUser(name string, fill bool) (*User, error) {
	user := &User{}
	if ID, found := usernameToID.Get(name); found {
		// Don't ping DB to fill the object
		if !fill {
			user.ID = ID.(uint)
			return user, nil
		}
		return user, DB.First(user, ID.(uint)).Error
	}
	err := DB.First(user, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	usernameToID.Set(name, user.ID, cache.NoExpiration)
	return user, nil
}

// isUser check if a username exists in the system
func isUser(name string) (bool, error) {
	if _, found := usernameToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&User{}, "name = ?", name).Count(&count).Error
	return count != 0, err
}

// getUserSources returns user's sources associated with him
func getUserSources(user string) ([]Source, error) {
	sources := make([]Source, 0, 16)
	err := DB.Model(sources).
		Joins("JOIN user_sources on sources.id = user_sources.source_id").
		Joins("JOIN users on user_sources.user_id = users.id AND users.name = ?", user).
		Find(&sources).
		Error
	return sources, err
}
