package storage

import (
	"errors"

	"github.com/patrickmn/go-cache"
	"github.com/thecsw/katya/log"
	"github.com/thecsw/katya/utils"
	"gorm.io/gorm"
)

// CreateUser creates a user in the database
func CreateUser(name, pass string) error {
	found, err := IsUser(name)
	if found {
		log.Error("User already exists", err, log.Params{"user": name})
		return errors.New("user already exists")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error("Failed to check user existence", err, log.Params{"user": name})
		return err
	}
	err = DB.Create(&User{Name: name, Password: utils.ShaEncode(pass)}).Error
	if err != nil {
		return err
	}
	log.Format("Successfully created a new user", log.Params{"name": name})
	return nil
}

// GetUser gets a user from the database by the username
func GetUser(name string, fill bool) (*User, error) {
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
		return user, err
	}
	usernameToID.Set(name, user.ID, cache.NoExpiration)
	return user, nil
}

// IsUser check if a username exists in the system
func IsUser(name string) (bool, error) {
	if _, found := usernameToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&User{}, "name = ?", name).Count(&count).Error
	return count != 0, err
}

// GetUserSources returns user's sources associated with him
func GetUserSources(user string) ([]Source, error) {
	sources := make([]Source, 0, 16)
	err := DB.Model(sources).
		Joins("INNER JOIN user_sources on sources.id = user_sources.source_id").
		Joins("INNER JOIN users on user_sources.user_id = users.id AND users.name = ?", user).
		Find(&sources).
		Error
	return sources, err
}

// GetUserSourcesEnabled returns user's sources associated with him that are enabled
func GetUserSourcesEnabled(user string) ([]Source, error) {
	sources := make([]Source, 0, 16)
	err := DB.Model(sources).
		Joins("INNER JOIN user_sources on sources.id = user_sources.source_id").
		Joins("INNER JOIN user_sources_enabled on user_sources.source_id = user_sources_enabled.source_id").
		Joins("INNER JOIN users on user_sources_enabled.user_id = users.id AND users.name = ?", user).
		Find(&sources).
		Error
	return sources, err
}
