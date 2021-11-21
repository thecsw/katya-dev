package storage

import (
	"github.com/patrickmn/go-cache"
	"github.com/thecsw/katya/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	// DB is our global instance of *gorm.DB
	DB *gorm.DB

	// usernameToID maps a username to its DB ID
	usernameToID = cache.New(cache.NoExpiration, cache.NoExpiration)
	// sourceToID maps source name to its DB ID
	sourceToID = cache.New(cache.NoExpiration, cache.NoExpiration)
	// crawlerToID maps crawler name to its DB ID
	crawlerToID = cache.New(cache.NoExpiration, cache.NoExpiration)
	// urlToID maps a text url to its DB ID
	urlToID = cache.New(cache.NoExpiration, cache.NoExpiration)
)

// InitDB opens the database.
func InitDB(dsn string) error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: log.DBLogger,
	})
	if err != nil {
		return err
	}
	err = DB.AutoMigrate(&User{}, &Source{}, &Crawler{}, &Scrape{}, &Global{}, &Text{})
	if err != nil {
		log.Error("Failed to automatically migrate gorm tables!", err, log.Params{"DSN": dsn})
		return err
	}
	return nil
}

// CloseDB closes the database.
func CloseDB() error {
	db, err := DB.DB()
	if err != nil {
		return err
	}
	err = db.Close()
	if err != nil {
		return err
	}
	return nil
}
