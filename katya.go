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
	"github.com/thecsw/katya/log"
	"github.com/thecsw/katya/storage"
)

const (
	// LISTEN_ADDRESS is the port number of our HTTP router
	LISTEN_ADDRESS = ":32000"

	// CRAWLERS_DIR is the directory for our crawlers (spiders)
	CRAWLERS_DIR = "./chelsea/chelsea/spiders/"
	// LOGS_DIR is where we send the logs from our crawlers (spiders)
	LOGS_DIR = "./logs/"
	// SCRAPY_DIR is the home directory of our scrapy instance
	SCRAPY_DIR = "./chelsea/"

	// RESTClientCert is the location of HTTP router's certificate
	RESTClientCert string = "./certs/fullchain.pem"
	// RESTClientKey is the location of HTTP router's private key
	RESTClientKey string = "./certs/privkey.pem"

	// DB_HOST is the destination address of our DB
	DB_HOST = "katya.sandyuraz.com"
	// DB_PORT is the DB port that we have
	DB_PORT = 5432
	// DB_NAME is the database name of our DB (usually username)
	DB_NAME = "sandy"
	// DB_USER is the DB user we will be working as
	DB_USER = "sandy"
	// DB_SSLMODE dictates on how we check our SSL
	DB_SSLMODE = "verify-full"
	// DB_SSLCERT is the certificate CA signed for us
	DB_SSLCERT = "./tools/client/client.crt"
	// DB_SSLKEY is our private key to prove our identity
	DB_SSLKEY = "./tools/client/client.key"
	// DB_SSLROOTCERT is the certificate list of the ruling CA (self-CA)
	DB_SSLROOTCERT = "./tools/ca/ca.crt"
)

var (
	//go:embed chelsea/chelsea/spiders/template.py
	templateCrawler string

	// dsn = fmt.Sprintf(
	// 	"host=%s port=%d user=%s dbname=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
	// 	HOST, PORT, USER, NAME, SSLMODE, SSLCERT, SSLKEY, SSLROOTCERT,
	// )

	// // dsn to connect to Postgres.
	dsn = "host=127.0.0.1 port=5432 user=sandy dbname=sandy"
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
	log.Init()

	// +-------------------------------------+
	// |              DATABASE               |
	// +-------------------------------------+

	// Initializing the database connection
	log.Format("Initializing the database connection", log.Params{"DSN": dsn})
	if err := storage.InitDB(dsn); err != nil {
		log.Error("Failed opening a database connection", err, log.Params{"DSN": dsn})
		return
	}
	defer func() {
		log.Format("Closing the database connection", log.Params{"DSN": dsn})
		storage.CloseDB()
	}()

	// +-------------------------------------+
	// |             OTHER STUFF             |
	// +-------------------------------------+
	log.Info("Checking for the existence of the global element")
	if !storage.DoesGlobalExist() {
		log.Info("Creating the global element")
		if err := storage.CreateGlobal(); err != nil {
			log.Error("failed to create a global element", err, log.Params{})
		}
	}
	// Create the delta caches
	log.Info("Creating delta words/sents caches")
	globalNumWordsDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)
	globalNumSentsDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)

	log.Info("Spinning up the words/sents goroutines")
	go updateGlobalWordSentsDeltas()
	go updateSourcesWordSentsDeltas()

	// Loading stopwords
	log.Info("Loading stopwords")
	loadStopwords()

	// +-------------------------------------+
	// |             HTTP Router             |
	// +-------------------------------------+

	log.Info("Creating our HTTP (API) router")
	myRouter := mux.NewRouter()
	myRouter.Use(basicMiddleware)

	myRouter.HandleFunc("/", helloReceiver).Methods(http.MethodGet)
	myRouter.HandleFunc("/noor", noorReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/status", statusReceiver).Methods(http.MethodPost)

	subRouter := myRouter.PathPrefix("/api").Subrouter()

	subRouter.HandleFunc("/auth", verifyAuth).Methods(http.MethodPost)
	subRouter.HandleFunc("/find", findQueryInTexts).Methods(http.MethodGet)
	subRouter.HandleFunc("/freqs", frequencyFinder).Methods(http.MethodGet)
	subRouter.HandleFunc("/trigger", crawlerRunner).Methods(http.MethodPost)
	subRouter.HandleFunc("/sources", userGetSources).Methods(http.MethodGet)
	subRouter.HandleFunc("/allocate", crawlerCreator).Methods(http.MethodPost)
	subRouter.HandleFunc("/source", userCreateSource).Methods(http.MethodPost)
	subRouter.HandleFunc("/source", userDeleteSource).Methods(http.MethodDelete)
	subRouter.HandleFunc("/status", crawlerStatusReceiver).Methods(http.MethodGet)

	log.Info("Enabled the auth portal for the API router")
	subRouter.Use(loggingMiddleware)

	// +-------------------------------------+
	// |              BLOCKING               |
	// +-------------------------------------+

	// Declare and define our HTTP handler
	log.Info("Configuring the HTTP router")
	corsOptions := cors.New(cors.Options{
		AllowedOrigins:     []string{"https://sandyuraz.com", "https://katya-kappa.vercel.app"},
		AllowedMethods:     []string{http.MethodPost, http.MethodGet, http.MethodDelete},
		AllowedHeaders:     []string{"Authorization", "Content-Type", "Access-Control-Allow-Methods"},
		ExposedHeaders:     []string{},
		MaxAge:             0,
		AllowCredentials:   true,
		OptionsPassthrough: false,
		Debug:              false,
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
			log.Error("Failed to fire up the router", err, log.Params{})
		}
	}()
	log.Info("Started the HTTP router, port: " + LISTEN_ADDRESS)

	//OLD SERVER
	// handler := cors.Default().Handler(myRouter)
	// srv := &http.Server{
	// 	Handler: handler,
	// 	Addr:    LISTEN_ADDRESS,
	// 	// Good practice: enforce timeouts for servers you create!
	// 	WriteTimeout: 15 * time.Second,
	// 	ReadTimeout:  15 * time.Second,
	// 	IdleTimeout:  60 * time.Second,
	// }
	// //Fire up the router
	// go func() {
	// 	if err := srv.ListenAndServe(); err != nil {
	// 		log.Info(err)
	// 	}
	// }()

	// Listen to SIGINT and other shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Info("API is shutting down")
}
