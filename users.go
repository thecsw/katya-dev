package main

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
)

// userCreateSource is our API endpoint to create a source for a user
func userCreateSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		lerr("Failed decoding a create source payload", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	user := r.Context().Value(ContextKey("user")).(User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = createSource(user.Name, payload.Link)
	if err != nil {
		lerr("Failed creating a source in http", err, params{})
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
		lerr("Failed decoding a create source payload", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, errors.Wrap(err, "bad request payload"))
		return
	}
	user := r.Context().Value(ContextKey("user")).(User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = removeSource(user.Name, payload.Link)
	if err != nil {
		lerr("Failed deleting a user source in http", err, params{})
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source deleted"}, http.StatusOK, nil)
}

// userGetSources is an API endpoint to return user's sources
func userGetSources(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(ContextKey("user")).(User)
	sources, err := getUserSources(user.Name)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "failed to retrieve sources"))
		return
	}
	httpJSON(w, sources, http.StatusOK, nil)
}
