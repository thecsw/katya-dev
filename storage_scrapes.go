package main

import "time"

// createScrape creates a scrape in the database
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

// finishScrape updates the crawler's latest scrape with an end and elapsed time
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

// getLastScrape returns the last registered scrape of the given crawler
func getLastScrape(crawlerName string) (*Scrape, error) {
	crawler, err := getCrawler(crawlerName, false)
	if err != nil {
		return nil, err
	}
	result := &Scrape{}
	return result, DB.Last(result, "crawler_id = ?", crawler.ID).Error
}
