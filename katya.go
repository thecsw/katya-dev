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
	LISTEN_ADDRESS = ":32000"

	CRAWLERS_DIR = "./chelsea/chelsea/spiders/"
	LOGS_DIR     = "./logs/"
	SCRAPY_DIR   = "./chelsea/"

	RESTClientCert string = "./certs/cert.pem"
	RESTClientKey  string = "./certs/privkey.pem"
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
	// Create the delta caches
	l("Creating delta words/sents caches")
	globalNumWordsDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)
	globalNumSentsDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)

	l("Creating default users users")
	createUser("sandy", "password")
	createUser("sandy2", "password")
	createUser("sandy3", "password")
	createUser("sandy4", "password")

	l("Spinning up the words/sents goroutines")
	go updateGlobalWordSentsDeltas()
	go updateSourcesWordSentsDeltas()

	// +-------------------------------------+
	// |             HTTP Router             |
	// +-------------------------------------+

	l("Creating our HTTP (API) router")
	myRouter := mux.NewRouter()

	myRouter.HandleFunc("/noor", noorReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/status", statusReceiver).Methods(http.MethodPost)

	subRouter := myRouter.PathPrefix("/api").Subrouter()

	subRouter.HandleFunc("/auth", verifyAuth).Methods(http.MethodPost)
	subRouter.HandleFunc("/find", textSearcher).Methods(http.MethodGet)
	subRouter.HandleFunc("/trigger", crawlerRunner).Methods(http.MethodPost)
	subRouter.HandleFunc("/sources", userGetSources).Methods(http.MethodGet)
	subRouter.HandleFunc("/allocate", crawlerCreator).Methods(http.MethodPost)
	subRouter.HandleFunc("/source", userCreateSource).Methods(http.MethodPost)
	subRouter.HandleFunc("/source", userDeleteSource).Methods(http.MethodDelete)
	subRouter.HandleFunc("/status", crawlerStatusReceiver).Methods(http.MethodGet)

	l("Enabled the auth portal for the API router")
	subRouter.Use(loggingMiddleware)

	// +-------------------------------------+
	// |              BLOCKING               |
	// +-------------------------------------+

	// Declare and define our HTTP handler
	l("Configuring the HTTP router")
	corsOptions := cors.New(cors.Options{
		//AllowedOrigins:   []string{"https://sandyuraz.com"},
		AllowedOrigins: []string{},
		AllowedMethods: []string{
			http.MethodPost,
			http.MethodGet,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            false,
	})
	handler := corsOptions.Handler(myRouter)
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
		// if err := srv.ListenAndServe(); err != nil {
		// 	log.Println(err)
		// }
		if err := srv.ListenAndServeTLS(RESTClientCert, RESTClientKey); err != nil {
			lerr("Failed to fire up the router", err, params{})
		}
	}()
	l("Started the HTTP router")

	// Listen to SIGINT and other shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	l("API is shutting down")
}
