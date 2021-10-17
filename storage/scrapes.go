package storage

import (
	"time"

	"github.com/thecsw/katya/log"
)

// CreateScrape creates a scrape in the database
func CreateScrape(crawlerName string) error {
	crawler, err := GetCrawler(crawlerName, false)
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
	log.Format("Successfully created a new scrape", log.Params{"crawler": crawlerName})
	return nil
}

// FinishScrape updates the crawler's latest scrape with an end and elapsed time
func FinishScrape(crawlerName string) error {
	crawler, err := GetCrawler(crawlerName, false)
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
	log.Format("Successfully finished a scrape", log.Params{"crawler": crawlerName})
	return nil
}

// GetLastScrape returns the last registered scrape of the given crawler
func GetLastScrape(crawlerName string) (*Scrape, error) {
	crawler, err := GetCrawler(crawlerName, false)
	if err != nil {
		return nil, err
	}
	result := &Scrape{}
	return result, DB.Last(result, "crawler_id = ?", crawler.ID).Error
}
