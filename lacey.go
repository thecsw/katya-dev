package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"github.com/thecsw/katya/log"
	"github.com/thecsw/katya/storage"
	"github.com/thecsw/katya/utils"
	"gorm.io/gorm"
)

// genCrawlerName takes a user and their link and returns a *guaranteed*
// unique name for a new crawler
func genCrawlerName(user, link string) string {
	return user + "-" + utils.ShaEncode(link)[:10]
}

// allocateCrawler actually tries to fully allocate and write a new crawler to disk
func allocateCrawler(user, link string, onlySubpaths bool) (string, error) {
	// Our name is going to be some UUID
	name := genCrawlerName(user, link)

	// Create params for logging purposes
	thisParams := log.Params{
		"name":          name,
		"user":          user,
		"url":           link,
		"only_subpaths": onlySubpaths,
	}

	// Check if a crawler already exists, if it doesn't,
	// then create one and use it later to trigger it
	crawlerExists, err := storage.IsCrawler(name)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error("failed existence allocating", err, thisParams)
		return "", errors.Wrap(err, "failed existence allocating")
	}

	// Create a crawler if one doesn't exist
	if !crawlerExists {
		err = storage.CreateCrawler(name, user, link)
		if err != nil {
			log.Error("failed creating allocating", err, thisParams)
			return "", errors.Wrap(err, "failed creating allocating")
		}
	}
	// Parse the url to retrieve domain
	parsedURL, err := url.Parse(link)
	if err != nil {
		log.Error("bad parsing of the source", err, thisParams)
		return "", errors.Wrap(err, "bad parsing of the source")
	}

	// Get the actual domain
	domain := parsedURL.Host

	// Write the scrapy python text file
	err = writeNewClawer(name, domain, link, onlySubpaths)
	if err != nil {
		log.Error("failed to write a crawler script", err, thisParams)
		return "", errors.Wrap(err, "failed to write a crawler script")
	}
	log.Format("Successfully allocated a new crawler", thisParams)
	return name, nil
}

// triggerCrawler actually triggers the saved crawler to start feeding Noor
func triggerCrawler(user, link string, logOutput io.Writer) (string, error) {
	// Our name is going to be some UUID
	name := genCrawlerName(user, link)
	// Create params for logging purposes
	thisParams := log.Params{
		"name": name,
		"user": user,
		"url":  link,
	}
	// Check if a crawler already exists, if it doesn't,
	// then create one and use it later to trigger it
	crawlerExists, err := storage.IsCrawler(name)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error("failed existence allocating", err, thisParams)
		return "", errors.Wrap(err, "failed existence allocating")
	}
	if !crawlerExists {
		err := errors.New("this crawler needs to be allocated first")
		log.Error("this crawler needs to be allocated first", err, thisParams)
		return "", err
	}
	log.Format("Triggering a crawler", thisParams)

	scrapyCmd := exec.Command("scrapy", "crawl", name)
	scrapyCmd.Dir = SCRAPY_DIR

	logFile, err := os.Create(LOGS_DIR + name + ".log")
	if err != nil {
		log.Error("Couldn't create a log file for a new scraper. Logs will be lost", err, thisParams)
		return "", err
	}

	logWriter := bufio.NewWriter(logFile)
	scrapyCmd.Stdout = logWriter
	scrapyCmd.Stderr = logWriter

	// Run the process in the background
	go func() {
		scrapyCmd.Run()
		// close the file
		logFile.Close()
	}()

	// Run is blocking, Start is non-blocking
	//return name, scrapyCmd.Run()
	return name, nil
}

// writeNewClawer copies the template scrapy python script into our own
// and updates its settings, like only subpaths or not
func writeNewClawer(name, domain, url string, onlySubpaths bool) error {
	myCrawler := templateCrawler

	myCrawler = strings.ReplaceAll(myCrawler, "<NAME>", name)
	myCrawler = strings.ReplaceAll(myCrawler, "<DOMAIN>", domain)
	myCrawler = strings.ReplaceAll(myCrawler, "<START>", url)
	if onlySubpaths {
		myCrawler = strings.ReplaceAll(myCrawler, "# nosubpath", "")
	}

	return ioutil.WriteFile(
		CRAWLERS_DIR+name+".py",
		[]byte(myCrawler),
		0600,
	)
}
