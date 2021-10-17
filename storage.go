package main

import (
	"github.com/patrickmn/go-cache"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	// HOST is the destination address of our DB
	HOST = "katya.sandyuraz.com"
	// PORT is the DB port that we have
	PORT = 5432
	// DBNAME is the database name of our DB (usually username)
	DBNAME = "sandy"
	// USER is the DB user we will be working as
	USER = "sandy"
	// SSLMODE dictates on how we check our SSL
	SSLMODE = "verify-full"
	// SSLCERT is the certificate CA signed for us
	SSLCERT = "./tools/client/client.crt"
	// SSLKEY is our private key to prove our identity
	SSLKEY = "./tools/client/client.key"
	// SSLROOTCERT is the certificate list of the ruling CA (self-CA)
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

	// usernameToID maps a username to its DB ID
	usernameToID = cache.New(cache.NoExpiration, cache.NoExpiration)
	// sourceToID maps source name to its DB ID
	sourceToID = cache.New(cache.NoExpiration, cache.NoExpiration)
	// crawlerToID maps crawler name to its DB ID
	crawlerToID = cache.New(cache.NoExpiration, cache.NoExpiration)
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
