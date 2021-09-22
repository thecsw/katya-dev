package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/pterm/pterm"
	"github.com/rs/cors"
)

const (
	LISTEN_ADDRESS = "0.0.0.0:10000"

	CRAWLERS_DIR = "./chelsea/chelsea/spiders/"
)

var (
	//go:embed chelsea/chelsea/spiders/template.py
	templateCrawler string
)

func main() {
	// +-------------------------------------+
	// |                INIT                 |
	// +-------------------------------------+

	// Print the big banner
	fmt.Println()
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
	l("Checking for the existence of the global element")
	if !doesGlobalExist() {
		l("Creating the global element")
		createGlobal()
	}
	globalNumWordsDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)
	globalNumSentsDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)

	l("Creating sandy user")
	createUser("sandy", "password")

	go updateGlobalWordSentsDeltas()
	go updateSourcesWordSentsDeltas()

	// createSource("sandy", "https://sandyuraz.com")
	// allocateCrawler("sandy", "https://sandyuraz.com", true)
	// createSource("sandy", "https://ilibrary.ru/text/1199")
	// allocateCrawler("sandy", "https://ilibrary.ru/text/1199", true)

	// +-------------------------------------+
	// |             HTTP Router             |
	// +-------------------------------------+

	l("Creating our HTTP router")
	myRouter := mux.NewRouter()

	myRouter.HandleFunc("/noor", noorReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/trigger", crawlerRunner).Methods(http.MethodPost)
	myRouter.HandleFunc("/status", statusReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/allocate", crawlerCreator).Methods(http.MethodPost)
	myRouter.HandleFunc("/source", userCreateSource).Methods(http.MethodPost)

	myRouter.HandleFunc("/find", textSearcher).Methods(http.MethodGet)
	myRouter.HandleFunc("/status", crawlerStatusReceiver).Methods(http.MethodGet)

	// +-------------------------------------+
	// |              BLOCKING               |
	// +-------------------------------------+

	// Declare and define our HTTP handler
	l("Configuring the HTTP router")
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
	l("Started the HTTP router")

	//fmt.Println(triggerCrawler("sandy", "https://sandyuraz.com", os.Stderr))
	//fmt.Println(triggerCrawler("sandy", "https://ilibrary.ru/text/1199", os.Stderr))
	// Listen to SIGINT and other shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	l("API is shutting down")
}
