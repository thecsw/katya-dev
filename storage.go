package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	HOST        = "127.0.0.1"
	PORT        = 5432
	DBNAME      = "sandy"
	USER        = "sandy"
	SSLMODE     = "verify-full"
	SSLCERT     = "./postgres/client.crt"
	SSLKEY      = "./postgres/client.key"
	SSLROOTCERT = "./postgres/ca.crt"
	tempRange   = 60
)

var (
	// DB is our global instance of *gorm.DB
	DB *gorm.DB

	// dsn = fmt.Sprintf(
	// 	"host=%s port=%d user=%s dbname=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
	// 	HOST, PORT, USER, DBNAME, SSLMODE, SSLCERT, SSLKEY, SSLROOTCERT,
	// )

	// dsn to connect to Postgres.
	dsn = fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s",
		HOST, PORT, USER, DBNAME,
	)

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

func createUser(name, pass string) error {
	err := DB.Create(&User{Name: name, Password: shaEncode(pass)}).Error
	if err != nil {
		return err
	}
	lf("Successfully created a new user", params{"name": name})
	return nil
}

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

func isUser(name string) (bool, error) {
	if _, found := usernameToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&User{}, "name = ?", name).Count(&count).Error
	return count != 0, err
}

func createGlobal() error {
	return DB.Create(&Global{NumWords: uint(0)}).Error
}

func doesGlobalExist() bool {
	count := int64(0)
	DB.First(&Global{}).Count(&count)
	return count != 0
}

func updateGlobal(numWords int) error {
	obj := &Global{}
	err := DB.First(obj).Error
	if err != nil {
		return err
	}
	obj.NumWords += uint(numWords)
	return DB.Save(obj).Error
}

func createSource(user, link string) error {
	userID, err := getUser(user, false)
	if err != nil {
		return err
	}
	toAdd := &Source{
		Link:     link,
		NumWords: 0,
	}
	err = DB.Model(userID).Association("Sources").Append(toAdd)
	if err != nil {
		return err
	}
	lf("Successfully created a new source", params{"user": user, "link": link})
	return nil
}

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
	err := DB.First(sourceObj, "link = ?", source).Error
	if err != nil {
		return nil, err
	}
	sourceToID.Set(source, sourceObj.ID, cache.NoExpiration)
	return sourceObj, nil
}

func updateSource(url string, numWords int) error {
	// Increase the number of words in the source cell
	source, err := getSource(url, true)
	if err != nil {
		return err
	}
	source.NumWords += uint(numWords)
	return DB.Save(source).Error
}

func isSource(name string) (bool, error) {
	if _, found := sourceToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&Source{}, "link = ?", name).Count(&count).Error
	return count != 0, err
}

func createCrawler(name, user, source string) error {
	userObj, err := getUser(user, false)
	if err != nil {
		return err
	}
	sourceObj, err := getSource(source, false)
	if err != nil {
		return err
	}
	err = DB.Create(&Crawler{
		Name:     name,
		SourceID: sourceObj.ID,
		UserID:   userObj.ID,
	}).Error
	if err != nil {
		return err
	}
	lf("Successfully created a new crawler", params{
		"name":   name,
		"user":   user,
		"source": source,
	})
	return nil
}

func getCrawler(name string, fill bool) (*Crawler, error) {
	crawlerObj := &Crawler{}
	if ID, found := crawlerToID.Get(name); found {
		// Don't ping DB to fill the object
		if !fill {
			crawlerObj.ID = ID.(uint)
			return crawlerObj, nil
		}
		return crawlerObj, DB.First(crawlerObj, ID.(uint)).Error
	}
	err := DB.First(crawlerObj, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	crawlerToID.Set(name, crawlerObj.ID, cache.NoExpiration)
	return crawlerObj, nil
}

func isCrawler(name string) (bool, error) {
	if _, found := crawlerToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&Crawler{}, "name = ?", name).Count(&count).Error
	return count != 0, err
}

func createScrape(crawlerName string) error {
	crawler, err := getCrawler(crawlerName, false)
	if err != nil {
		return err
	}
	err = DB.Create(&Scrape{
		CrawlerID: crawler.ID,
		Start:     uint(time.Now().Unix()),
		Elapsed:   0,
		End:       0,
	}).Error
	if err != nil {
		return err
	}
	lf("Successfully created a new scrape", params{"crawler": crawlerName})
	return nil
}

func finishScrape(crawlerName string) error {
	crawler, err := getCrawler(crawlerName, false)
	if err != nil {
		return err
	}
	scrape := &Scrape{}
	err = DB.Last(scrape, "crawler_id = ?", crawler.ID).Error
	if err != nil {
		return err
	}
	scrape.End = uint(time.Now().Unix())
	scrape.Elapsed = scrape.End - scrape.Start
	err = DB.Save(scrape).Error
	if err != nil {
		return err
	}
	lf("Successfully finished a scrape", params{"crawler": crawlerName})
	return nil
}

func getLastScrape(crawlerName string) (*Scrape, error) {
	crawler, err := getCrawler(crawlerName, false)
	if err != nil {
		return nil, err
	}
	result := &Scrape{}
	return result, DB.Last(result, "crawler_id = ?", crawler.ID).Error
}

func createText(
	source string,
	url string,
	ip string,
	status uint,
	text string,
	title string,
	numWords uint,
) error {
	sourceObj, err := getSource(source, false)
	if err != nil {
		return err
	}
	toAdd := &Text{
		URL:      url,
		IP:       ip,
		Status:   status,
		Text:     text,
		Title:    title,
		NumWords: numWords,
	}
	err = DB.Model(sourceObj).Association("Texts").Append(toAdd)
	if err != nil {
		return err
	}
	lf("Successfully created a new text", params{
		"url":       url,
		"title":     title,
		"ip":        ip,
		"num_words": numWords,
	})
	return nil
}

func findTexts(query string, limit int, offset int) ([]Text, error) {
	texts := make([]Text, 0, limit)
	err := DB.
		Where(
			"lower(text) LIKE ?",
			"%"+strings.ToLower(query)+"%",
		).
		Limit(limit).
		Offset(offset).
		Find(&texts).
		Error
	return texts, err
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// reverseString from https://groups.google.com/g/golang-nuts/c/oPuBaYJ17t4
func reverseString(what string) string {
	// Get Unicode code points.
	n := 0
	rune := make([]rune, len(what))
	for _, r := range what {
		rune[n] = r
		n++
	}
	rune = rune[0:n]
	// Reverse
	for i := 0; i < n/2; i++ {
		rune[i], rune[n-1-i] = rune[n-1-i], rune[i]
	}
	// Convert back to UTF-8.
	return string(rune)
}
