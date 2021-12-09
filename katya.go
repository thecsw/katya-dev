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
	// development environment keys
	katyaDevEnvironment = true

	// ListenAddress is the port number of our HTTP router
	ListenAddress = ":32000"

	// CrawlersDir is the directory for our crawlers (spiders)
	CrawlersDir = "./scrapy/katya_crawlers/spiders/"
	// LogsDir is where we send the logs from our crawlers (spiders)
	LogsDir = "./logs/"
	// ScrapyDir is the home directory of our scrapy instance
	ScrapyDir = "./scrapy/"

	// RESTClientCert is the location of HTTP router's certificate
	RESTClientCert string = "./certs/fullchain.pem"
	// RESTClientKey is the location of HTTP router's private key
	RESTClientKey string = "./certs/privkey.pem"

	// DbHost is the destination address of our DB
	DbHost = "katya-api.sandyuraz.com"
	// DbPort is the DB port that we have
	DbPort = 5432
	// DbName is the database name of our DB (usually username)
	DbName = "sandy"
	// DbUser is the DB user we will be working as
	DbUser = "sandy"
	// DbSSLMode dictates on how we check our SSL
	DbSSLMode = "verify-full"
	// DbSSLCertificate is the certificate CA signed for us
	DbSSLCertificate = "./tools/client/client.crt"
	// DbSSLKey is our private key to prove our identity
	DbSSLKey = "./tools/client/client.key"
	// DbSSLRootCertificate is the certificate list of the ruling CA (self-CA)
	DbSSLRootCertificate = "./tools/ca/ca.crt"
)

var (
	// banner to show below the katya text banner
	katyaBannerStrip = "Production Environment"

	//go:embed data/template_spider.py
	templateCrawler string

	// dsn to connect to Postgres.
	dsn = fmt.Sprintf(
		"host=%s port=%d user=%s dbname=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s",
		DbHost, DbPort, DbUser, DbName, DbSSLMode, DbSSLCertificate, DbSSLKey, DbSSLRootCertificate,
	)

	// HTTP stuff for CORS pre-flight requests
	allowedOrigins = []string{"https://katya.sandyuraz.com", "https://katya-kappa.vercel.app"}
	allowedMethods = []string{http.MethodPost, http.MethodGet, http.MethodDelete}
	allowedHeaders = []string{"Authorization", "Content-Type", "Access-Control-Allow-Methods"}
)

func main() {
	// Enable debug environment if the global flag is true
	if katyaDevEnvironment {
		dsn = "host=127.0.0.1 port=5432 user=sandy dbname=sandy"
		katyaBannerStrip = "Development Environment"

		allowedOrigins = append(allowedOrigins, "http://localhost:5000")
	}

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

	pterm.DefaultCenter.Print(
		pterm.DefaultHeader.
			WithFullWidth().
			WithBackgroundStyle(pterm.NewStyle(pterm.BgBlack)).
			WithMargin(1).Sprint(katyaBannerStrip))

	// Initialize our log instance
	log.Init()

	// Initializing the database connection
	log.Format("Initializing the database connection", log.Params{"DSN": dsn})
	if err := storage.InitDB(dsn); err != nil {
		log.Error("Failed opening a database connection", err, log.Params{"DSN": dsn})
		return
	}
	defer func() {
		log.Format("Closing the database connection", log.Params{"DSN": dsn})
		err := storage.CloseDB()
		if err != nil {
			log.Error("Failed closing the database connection", err, log.Params{"DSN": dsn})
			return
		}
	}()

	// +-------------------------------------+
	// |             OTHER STUFF             |
	// +-------------------------------------+
	log.Info("Checking for the existence of the global element")
	if !storage.DoesGlobalExist() {
		log.Info("Creating the global element")
		if err := storage.CreateGlobal(); err != nil {
			log.Error("failed to create a global element", err, nil)
		}
	}
	// Create the delta caches
	log.Info("Creating delta words/sentences caches")
	_ = globalNumWordsDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)
	_ = globalNumSentencesDelta.Add(globalDeltaCacheKey, uint(0), cache.NoExpiration)

	log.Info("Spinning up the words/sentences goroutines")
	go func() {
		for {
			time.Sleep(deltaUpdateInterval)
			updateGlobalWordSentencesDeltas()
		}
	}()
	go func() {
		for {
			time.Sleep(deltaUpdateInterval)
			updateSourcesWordSentencesDeltas()
		}
	}()

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
	myRouter.HandleFunc("/text", textReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/status", statusReceiver).Methods(http.MethodPost)

	subRouter := myRouter.PathPrefix("").Subrouter()

	subRouter.HandleFunc("/auth", verifyAuth).Methods(http.MethodPost)
	subRouter.HandleFunc("/find", findQueryInTexts).Methods(http.MethodGet)
	subRouter.HandleFunc("/trigger", crawlerRunner).Methods(http.MethodPost)
	subRouter.HandleFunc("/sources", userGetSources).Methods(http.MethodGet)
	subRouter.HandleFunc("/allocate", crawlerCreator).Methods(http.MethodPost)
	subRouter.HandleFunc("/source", userCreateSource).Methods(http.MethodPost)
	subRouter.HandleFunc("/source", userDeleteSource).Methods(http.MethodDelete)
	subRouter.HandleFunc("/frequencies", frequencyFinder).Methods(http.MethodGet)
	subRouter.HandleFunc("/status", crawlerStatusReceiver).Methods(http.MethodGet)

	log.Info("Enabled the auth portal for the API router")
	subRouter.Use(loggingMiddleware)

	// Final preparations for the dev environment if enabled
	if katyaDevEnvironment {
		// Create a default user
		storage.CreateUser("sandy", "urazayev")
	}

	// Declare and define our HTTP handler
	log.Info("Configuring the HTTP router")
	corsOptions := cors.New(cors.Options{
		AllowedOrigins:     allowedOrigins,
		AllowedMethods:     allowedMethods,
		AllowedHeaders:     allowedHeaders,
		ExposedHeaders:     []string{},
		MaxAge:             0,
		AllowCredentials:   true,
		OptionsPassthrough: false,
		Debug:              false,
	})
	handler := corsOptions.Handler(myRouter)
	srv := &http.Server{
		Handler: handler,
		Addr:    ListenAddress,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	// Fire up the router
	go func() {
		if katyaDevEnvironment {
			if err := srv.ListenAndServe(); err != nil {
				log.Error("Failed to fire up the router", err, nil)
			}
		} else {
			if err := srv.ListenAndServeTLS(RESTClientCert, RESTClientKey); err != nil {
				log.Error("Failed to fire up the router", err, nil)
			}
		}
	}()
	log.Info("Started the HTTP router, port " + ListenAddress)

	// Listen to SIGINT and other shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	log.Info("API is shutting down")
	// Run the final updates
	log.Info("Flushing last delta updates")
	updateGlobalWordSentencesDeltas()
	updateSourcesWordSentencesDeltas()
}
