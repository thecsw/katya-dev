package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"net/http"
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

// crawlerActionPayload is the POST body of crawler actions
type crawlerActionPayload struct {
	// Link is the crawler's link
	Link string `json:"link"`
	// Label is just user-created custom text
	Label string `json:"label"`
	// Enabled flags if something is disabled (defaults to false -> enabled)
	Disabled bool `json:"disabled"`
	// OnlySubpaths tells us if we only do subdirectories of the link
	OnlySubpaths bool `json:"only_subpaths"`
}

// crawlerCreator creates a crawler
func crawlerCreator(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a crawler creator payload", err, nil)
		return
	}
	user := r.Context().Value(ContextKey("user")).(storage.User)
	thisLogParams := log.Params{
		"user": user.Name,
		"link": payload.Link,
	}
	name, err := allocateCrawler(user.Name, payload.Link, payload.OnlySubpaths)
	if err != nil {
		log.Error("Failed allocating a crawler in creator payload", err, thisLogParams)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	httpJSON(w, httpMessageReturn{"created crawler: " + name}, http.StatusOK, nil)
}

// crawlerRunner triggers a crawler
func crawlerRunner(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a crawler trigger payload", err, nil)
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	user := r.Context().Value(ContextKey("user")).(storage.User)
	thisLogParams := log.Params{
		"user": user.Name,
		"link": payload.Link,
	}
	name, err := triggerCrawler(user.Name, payload.Link)
	if err != nil {
		log.Error("Failed triggering a crawler in creator payload", err, thisLogParams)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	httpJSON(w, httpMessageReturn{"triggered crawler: " + name}, http.StatusOK, nil)
}

func crawlerStatusReceiver(w http.ResponseWriter, r *http.Request) {
	crawlerName := r.URL.Query().Get("name")
	if crawlerName == "" {
		httpJSON(w, nil, http.StatusBadRequest, errors.New("empty crawler name"))
		return
	}
	val, err := storage.GetLastScrape(crawlerName)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "getting last scrape"))
		return
	}
	httpJSON(w, *val, http.StatusOK, nil)
}

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
	err = writeNewCrawler(name, domain, link, onlySubpaths)
	if err != nil {
		log.Error("failed to write a crawler script", err, thisParams)
		return "", errors.Wrap(err, "failed to write a crawler script")
	}
	log.Format("Successfully allocated a new crawler", thisParams)
	return name, nil
}

// triggerCrawler actually triggers the saved crawler to start feeding texts
func triggerCrawler(user, link string) (string, error) {
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
	scrapyCmd.Dir = ScrapyDir

	logFile, err := os.Create(LogsDir + name + ".log")
	if err != nil {
		log.Error("Couldn't create a log file for a new scraper. Logs will be lost", err, thisParams)
		return "", err
	}

	logWriter := bufio.NewWriter(logFile)
	scrapyCmd.Stdout = logWriter
	scrapyCmd.Stderr = logWriter

	// Run the process in the background
	go func() {
		err := scrapyCmd.Run()
		if err != nil {
			log.Error("FAILED TO START SCRAPY", err, log.Params{"log_file": logFile.Name()})
			return
		}
		// close the file
		err = logFile.Close()
		if err != nil {
			log.Error("FAILED CLOSE SCRAPY LOG FILE", err, log.Params{"log_file": logFile.Name()})
			return
		}
	}()

	// Run is blocking, Start is non-blocking
	//return name, scrapyCmd.Run()
	return name, nil
}

// writeNewCrawler copies the template scrapy python script into our own
// and updates its settings, like only subpaths or not
func writeNewCrawler(name, domain, url string, onlySubpaths bool) error {
	myCrawler := templateCrawler

	myCrawler = strings.ReplaceAll(myCrawler, "<NAME>", name)
	myCrawler = strings.ReplaceAll(myCrawler, "<DOMAIN>", domain)
	myCrawler = strings.ReplaceAll(myCrawler, "<START>", url)
	if onlySubpaths {
		myCrawler = strings.ReplaceAll(myCrawler, "# nosubpath", "")
	}

	return ioutil.WriteFile(
		CrawlersDir+name+".py",
		[]byte(myCrawler),
		0600,
	)
}
