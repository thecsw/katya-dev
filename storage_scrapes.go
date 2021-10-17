package main

import "time"

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
