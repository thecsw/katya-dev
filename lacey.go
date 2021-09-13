package main

import (
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func genCrawlerName(user, link string) string {
	return user + shaEncode(link)[:10]
}

func allocateCrawler(user, link string) error {
	// Our name is going to be some UUID
	name := genCrawlerName(user, link)
	// Create params for logging purposes
	thisParams := params{
		"name": name,
		"user": user,
		"url":  link,
	}
	// Check if a crawler already exists, if it doesn't,
	// then create one and use it later to trigger it
	crawlerExists, err := isCrawler(name)
	if err != nil && err != gorm.ErrRecordNotFound {
		lerr("failed existence allocating", err, thisParams)
		return errors.Wrap(err, "failed existence allocating")
	}
	// Create a crawler if one doesn't exist
	if !crawlerExists {
		err = createCrawler(name, user, link)
		if err != nil {
			lerr("failed creating allocating", err, thisParams)
			return errors.Wrap(err, "failed creating allocating")
		}
	}
	// Parse the url to retrieve domain
	parsedURL, err := url.Parse(link)
	if err != nil {
		lerr("bad parsing of the source", err, thisParams)
		return errors.Wrap(err, "bad parsing of the source")
	}
	domain := parsedURL.Host

	err = writeNewClawer(name, domain, link, true)
	if err != nil {
		lerr("failed to write a crawler script", err, thisParams)
		return errors.Wrap(err, "failed to write a crawler script")
	}
	lf("Successfully allocated a new crawler", thisParams)
	return nil
}

// func triggerCrawler(user, link string) error {
// 	// Our name is going to be some UUID
// 	name := genCrawlerName(user, link)
// 	// Create params for logging purposes
// 	thisParams := params{
// 		"name": name,
// 		"user": user,
// 		"url":  link,
// 	}
// 	// Check if a crawler already exists, if it doesn't,
// 	// then create one and use it later to trigger it
// 	crawlerExists, err := isCrawler(name)
// 	if err != nil && err != gorm.ErrRecordNotFound {
// 		lerr("failed existence allocating", err, thisParams)
// 		return errors.Wrap(err, "failed existence allocating")
// 	}
// 	if !crawlerExists {
// 		err := errors.New("this crawler needs to be allocated first")
// 		lerr("this crawler needs to be allocated first", err, thisParams)
// 		return err
// 	}
// 	scrapyCmd := exec.Command("scrapy", "crawl", name)
// }

func writeNewClawer(name, domain, url string, subpath bool) error {
	myCrawler := templateCrawler

	myCrawler = strings.ReplaceAll(myCrawler, "<NAME>", name)
	myCrawler = strings.ReplaceAll(myCrawler, "<DOMAIN>", domain)
	myCrawler = strings.ReplaceAll(myCrawler, "<START>", url)
	if subpath {
		myCrawler = strings.ReplaceAll(myCrawler, "#nosubpath", "")
	}

	return ioutil.WriteFile(
		CRAWLERS_DIR+name+".py",
		[]byte(myCrawler),
		0644,
	)
}
