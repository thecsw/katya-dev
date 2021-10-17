package main

import (
	"github.com/patrickmn/go-cache"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	HOST        = "katya.sandyuraz.com"
	PORT        = 5432
	DBNAME      = "sandy"
	USER        = "sandy"
	SSLMODE     = "verify-full"
	SSLCERT     = "./tools/client/client.crt"
	SSLKEY      = "./tools/client/client.key"
	SSLROOTCERT = "./tools/ca/ca.crt"
)

var (
	// DB is our global instance of *gorm.DB
	DB *gorm.DB

	// dsn = fmt.Sprintf(
	// 	"host=%s port=%d user=%s dbname=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
	// 	HOST, PORT, USER, DBNAME, SSLMODE, SSLCERT, SSLKEY, SSLROOTCERT,
	// )

	// // dsn to connect to Postgres.
	dsn = "host=127.0.0.1 port=5432 user=sandy dbname=sandy"

	// Couple of caches that we would use
	usernameToID = cache.New(cache.NoExpiration, cache.NoExpiration)
	sourceToID   = cache.New(cache.NoExpiration, cache.NoExpiration)
	crawlerToID  = cache.New(cache.NoExpiration, cache.NoExpiration)
)

// init opens the database.
func initDB() error {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: dbLogger,
	})
	if err != nil {
		return err
	}
	DB.AutoMigrate(&User{}, &Source{}, &Crawler{}, &Scrape{}, &Global{}, &Text{})
	return nil
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
