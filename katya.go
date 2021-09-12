package main

import (
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func main() {
	myRouter := mux.NewRouter()
	myRouter.HandleFunc("/noor", noorReceiver).Methods(http.MethodPost)
	myRouter.HandleFunc("/status", statusReceiver).Methods(http.MethodPost)

	listenAddr := "0.0.0.0:10000"
	log.Infof("Listening on %s... ", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, myRouter))
}
