package main

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	HOST        = "elephant.sandyuraz.com"
	PORT        = 5432
	DBNAME      = "sandissa"
	USER        = "sandy"
	SSLMODE     = "verify-full"
	SSLCERT     = "./postgres/client.crt"
	SSLKEY      = "./postgres/client.key"
	SSLROOTCERT = "./postgres/ca.crt"
	tempRange   = 60
)

var (
	DB *gorm.DB

	dsn = fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
		HOST, PORT, USER, DBNAME, SSLMODE, SSLCERT, SSLKEY, SSLROOTCERT,
	)
)

// init opens the database.
func initDB() error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	DB.AutoMigrate(&User{}, &Source{}, &Text{})
	return nil
}

func createUser(name string) error {
	return DB.Create(&User{Name: name}).Error
}

func createSource(user, link string) error {
	userID := uint(0) // find this
	return DB.Create(&Source{UserID: userID, Link: link}).Error
}

func createText(
	source string, spider string, url string,
	ip string, status int, text string) error {
	sourceID := uint(0) // find this
	return DB.Create(&Text{
		SourceID:  sourceID,
		NameCrawl: spider,
		URL:       url,
		IP:        ip,
		Status:    status,
		Text:      text,
	}).Error
}

// closeDB closes the database.
func closeDB() error {
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
