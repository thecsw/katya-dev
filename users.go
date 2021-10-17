package main

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/thecsw/katya/log"
	"github.com/thecsw/katya/storage"
)

// userCreateSource is our API endpoint to create a source for a user
func userCreateSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a create source payload", err, log.Params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	user := r.Context().Value(ContextKey("user")).(storage.User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = storage.CreateSource(user.Name, payload.Link)
	if err != nil {
		log.Error("Failed creating a source in http", err, log.Params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source created"}, http.StatusOK, nil)
}

// userDeleteSource is our API endpoint to delete a source for a user
func userDeleteSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a create source payload", err, log.Params{})
		httpJSON(w, nil, http.StatusBadRequest, errors.Wrap(err, "bad request payload"))
		return
	}
	user := r.Context().Value(ContextKey("user")).(storage.User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = storage.RemoveSource(user.Name, payload.Link)
	if err != nil {
		log.Error("Failed deleting a user source in http", err, log.Params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source deleted"}, http.StatusOK, nil)
}

// userGetSources is an API endpoint to return user's sources
func userGetSources(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(ContextKey("user")).(storage.User)
	sources, err := storage.GetUserSources(user.Name)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "failed to retrieve sources"))
		return
	}
	httpJSON(w, sources, http.StatusOK, nil)
}
