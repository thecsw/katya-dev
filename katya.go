package main

import (
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	LISTEN_ADDRESS = "0.0.0.0:10000"

	TEMPLATE_CRAWLER = "./chelsea/chelsea/spiders/template.py"
	CRAWLERS_DIR     = "./chelsea/chelsea/spiders/"
)

var (
	templateCrawler = ""
)

func main() {
	// +-------------------------------------+
	// |                INIT                 |
	// +-------------------------------------+

	// Initialize our log instance
	linit()
	templateCrawlerTemp, err := ioutil.ReadFile(TEMPLATE_CRAWLER)
	if err != nil {
		lerr("Failed opening the template crawler", err, params{})
		return
	}
	templateCrawler = string(templateCrawlerTemp)

	// +-------------------------------------+
	// |              DATABASE               |
	// +-------------------------------------+

	// Initializing the database connection
	lf("Initializing the database connection", params{"DSN": dsn})
	if err := initDB(); err != nil {
		lerr("Failed opening a database connection", err, params{"DSN": dsn})
		return
	}
	defer func() {
		lf("Closing the database connection", params{"DSN": dsn})
		closeDB()
	}()

	// +-------------------------------------+
	// |             OTHER STUFF             |
	// +-------------------------------------+

	l("Creating sandy user")
	createUser("sandy", "password")
	createSource("sandy", "https://sandyuraz.com")
	allocateCrawler("sandy", "https://sandyuraz.com")
	createSource("sandy", "https://ilibrary.ru/text/1199")
	allocateCrawler("sandy", "https://ilibrary.ru/text/1199")

	// l("Creating a new source")
	// createSource("sandy", "https://ilibrary.ru/text/1199")

	// +-------------------------------------+
	// |             HTTP Router             |
	// +-------------------------------------+

	l("Creating our HTTP router")
	myRouter := mux.NewRouter()
	myRouter.HandleFunc("/noor", noorReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/status", statusReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/find", textSearcher).Methods(http.MethodGet)

	// +-------------------------------------+
	// |              BLOCKING               |
	// +-------------------------------------+

	log.Infof("Listening on %s... ", LISTEN_ADDRESS)
	log.Fatal(http.ListenAndServe(LISTEN_ADDRESS, myRouter))
}
