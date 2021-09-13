package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/pterm/pterm"
	"github.com/rs/cors"
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

	// Print the big banner
	s, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("K", pterm.NewStyle(pterm.FgMagenta)),
		pterm.NewLettersFromStringWithStyle("atya", pterm.NewStyle(pterm.FgGreen)),
	).Srender()

	pterm.DefaultCenter.Print(s)
	pterm.DefaultCenter.
		WithCenterEachLineSeparately().
		Println("Katya and friends or The Liberated Corpus")

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
	if !doesGlobalExist() {
		createGlobal()
	}

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

	// Declare and define our HTTP handler
	handler := cors.Default().Handler(myRouter)
	srv := &http.Server{
		Handler: handler,
		Addr:    LISTEN_ADDRESS,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	// Fire up the router
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	l("Started the API router")
	// Listen to SIGINT and other shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	l("API is shutting down")

}
