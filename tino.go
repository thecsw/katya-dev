package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func noorReceiver(w http.ResponseWriter, r *http.Request) {
	payload := &NoorPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		fmt.Println("======================================")
	}
	fmt.Println(payload.URL)
	fmt.Println(payload.Text)

	w.WriteHeader(http.StatusOK)
}

type NoorPayload struct {
	Name     string `json:"name"`
	StartURL string `json:"start"`
	URL      string `json:"url"`
	IP       string `json:"ip"`
	Status   int    `json:"status"`
	Text     string `json:"text"`
}

func statusReceiver(w http.ResponseWriter, r *http.Request) {
	payload := &StatusPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		fmt.Println("======================================")
	}
	fmt.Println(payload.Status)

	w.WriteHeader(http.StatusOK)
}

type StatusPayload struct {
	Status string `json:"status"`
}
