package main

import (
	"errors"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func getNumOfSources() (uint, error) {
	count := uint(0)
	return count, DB.
		Raw("SELECT count(1) FROM sources").
		Scan(&count).
		Error
}

func updateGlobalWordNum(numWords uint) error {
	return DB.Exec(
		"UPDATE globals SET num_words = num_words + ? where id = 1",
		numWords).
		Error
}

func updateGlobalSentNum(numSents uint) error {
	return DB.Exec(
		"UPDATE globals SET num_sentences = num_sentences + ? where id = 1",
		numSents).
		Error
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

func getUserSources(user string) ([]Source, error) {
	sources := make([]Source, 0, 16)
	err := DB.Model(sources).
		Joins("JOIN user_sources on sources.id = user_sources.source_id").
		Joins("JOIN users on user_sources.user_id = users.id AND users.name = ?", user).
		Find(&sources).
		Error
	return sources, err
}

func updateSourceWordNum(url string, numWords uint) error {
	return DB.Exec(
		"UPDATE sources SET num_words = num_words + ? WHERE link = ?",
		numWords, url).
		Error
}

func updateSourceSentNum(url string, numSents uint) error {
	return DB.Exec(
		"UPDATE sources SET num_sentences = num_sentences + ? WHERE link = ?",
		numSents, url).
		Error
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
	original string,
	text string,
	shapes string,
	tags string,
	nomins string,
	title string,
	numWords uint,
	numSents uint,
) error {
	sourceObj, err := getSource(source, false)
	if err != nil {
		return err
	}
	toAdd := &Text{
		URL:         url,
		IP:          ip,
		Status:      status,
		Original:    original,
		Text:        text,
		Shapes:      shapes,
		Tags:        tags,
		Nominatives: nomins,
		Title:       title,
		NumWords:    numWords,
		NumSents:    numSents,
		Sources:     []*Source{},
	}
	err = DB.
		Model(sourceObj).
		Clauses(clause.OnConflict{
			DoNothing: true,
			UpdateAll: true,
		}).
		Association("Texts").
		Append(toAdd)
	if err != nil {
		if strings.Contains(err.Error(), "SQLSTATE 23503") {
			lf("Text link already exists, not replacing.", params{
				"url":   url,
				"title": title,
			})
			return errors.New("already exists")
		}
		return err
	}
	lf("Successfully created a new text", params{
		"url":       url,
		"title":     title,
		"ip":        ip,
		"num_words": numWords,
		"num_sents": numSents,
	})
	return nil
}

func findTexts(
	user string,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	texts := make([]Text, 0, limit)
	sqlWhere := "texts.text LIKE ?"
	sqlMatch := "%" + query + "%"
	if !caseSensitive {
		sqlWhere = "lower(texts.text) LIKE ?"
		sqlMatch = "%" + strings.ToLower(query) + "%"
	}
	err := DB.Model(texts).
		Joins("JOIN source_texts on texts.id = source_texts.text_id").
		Joins("JOIN sources on sources.id = source_texts.source_id").
		Joins("JOIN user_sources on sources.id = user_sources.source_id").
		Joins("JOIN users on user_sources.user_id = users.id AND users.name = ?", user).
		Where(sqlWhere, sqlMatch).
		Limit(limit).
		Offset(offset).
		Find(&texts).
		Error
	return texts, err
}

var (
	findByPart = map[string]func(uint, string, int, int, bool) ([]Text, error){
		"text":   findTextsByUserID,
		"shapes": findShapesByUserID,
		"tags":   findTagsByUserID,
		"nomins": findNominativesByUserID,
	}
)

func findTextsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.text", userID, query, limit, offset, caseSensitive)
}

func findShapesByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.shapes", userID, query, limit, offset, caseSensitive)
}

func findTagsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.tags", userID, query, limit, offset, caseSensitive)
}

func findNominativesByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.nominatives", userID, query, limit, offset, caseSensitive)
}

func findTextsPartsByUserID(
	part string,
	userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	texts := make([]Text, 0, limit)
	sqlWhere := part + " LIKE ?"
	sqlMatch := "%" + query + "%"
	if !caseSensitive {
		sqlWhere = "lower(" + part + ") LIKE ?"
		sqlMatch = "%" + strings.ToLower(query) + "%"
	}
	err := DB.Model(texts).
		Joins("INNER JOIN source_texts on texts.id = source_texts.text_id").
		Joins("INNER JOIN sources on sources.id = source_texts.source_id").
		Joins("INNER JOIN user_sources on sources.id = user_sources.source_id").
		Joins("INNER JOIN users on user_sources.user_id = users.id AND users.id = ?", userID).
		Where(sqlWhere, sqlMatch).
		Limit(limit).
		Offset(offset).
		Find(&texts).
		Error
	return texts, err
}

// max returns the max of its arguments
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the min of its arguments
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
