package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/thecsw/katya/log"
	"github.com/thecsw/katya/storage"
	"gorm.io/gorm"
)

// userCreateSource is our API endpoint to create a source for a user
func userCreateSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a create source payload", err, nil)
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	user := r.Context().Value(ContextKey("user")).(storage.User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	sourceExists, err := storage.IsSource(payload.Link)
	//litter.Dump(sourceExists)
	// Default label to its link
	if payload.Label == "" {
		payload.Label = payload.Link
	}
	// Update the label and enabled status of the source
	if sourceExists && err != gorm.ErrRecordNotFound {
		// Link the source if it's not already linked
		storage.AddSource(user.Name, payload.Link)
		err = storage.UpdateSourceLabel(payload.Link, payload.Label)
		if err != nil {
			log.Error("Failed updating source's label", err, nil)
			httpJSON(w, nil, http.StatusBadRequest, err)
			return
		}
		if payload.Disabled {
			storage.DisableSource(user.Name, payload.Link)
		} else {
			storage.EnableSource(user.Name, payload.Link)
		}
		httpJSON(w, httpMessageReturn{fmt.Sprintf("Label: %s, Enabled: %v", payload.Label, !payload.Disabled)}, http.StatusOK, nil)
		return
	}
	err = storage.CreateSource(user.Name, payload.Link, payload.Label)
	if err != nil {
		log.Error("Failed creating a source in http", err, nil)
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	// Automatically allocate a crawler
	crawlerName, err := allocateCrawler(user.Name, payload.Link, true)
	if err != nil {
		log.Error("Failed creating a source crawler in http", err, nil)
		httpJSON(w, nil, http.StatusBadRequest, err)
		return
	}
	httpJSON(w, httpMessageReturn{"source created with link: " + payload.Link + ", crawler: " + crawlerName}, http.StatusOK, nil)
}

// userDeleteSource is our API endpoint to delete a source for a user
func userDeleteSource(w http.ResponseWriter, r *http.Request) {
	payload := &crawlerActionPayload{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(payload)
	if err != nil {
		log.Error("Failed decoding a create source payload", err, nil)
		httpJSON(w, nil, http.StatusBadRequest, errors.Wrap(err, "bad request payload"))
		return
	}
	// check that the link value isn't empty
	if len(payload.Link) < 1 {
		httpJSON(w, nil, http.StatusBadRequest, errors.Wrap(err, "received empty link"))
		return
	}
	user := r.Context().Value(ContextKey("user")).(storage.User)
	// If our link is ending with a slash, remove it
	if payload.Link[len(payload.Link)-1] == '/' {
		payload.Link = payload.Link[:len(payload.Link)-1]
	}
	err = storage.RemoveSource(user.Name, payload.Link)
	if err != nil {
		err = errors.Wrap(err, "Failed deleting a user source in http")
		log.Error("", err, nil)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	err = storage.DisableSource(user.Name, payload.Link)
	if err != nil {
		err = errors.Wrap(err, "Failed deleting a user enabled source in http")
		log.Error("", err, nil)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	// Let's try to *hard* delete a source if there are no texts associated with it
	// and no users have the source associated with them either
	sourceID, err := storage.GetSource(payload.Link, false)
	if err != nil {
		err = errors.Wrap(err, "Failed retrieving source for hard delete check")
		log.Error("", err, nil)
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	numUsers, err1 := storage.GetSourcesUsers(sourceID.ID)
	numEnabledUsers, err2 := storage.GetSourcesEnabledUsers(sourceID.ID)
	if len(numUsers) == 0 && len(numEnabledUsers) == 0 && err1 == nil && err2 == nil {
		err = storage.HardDeleteSource(payload.Link)
		if err != nil {
			err = errors.Wrap(err, "Failed hard deleting")
			log.Error("", err, nil)
			httpJSON(w, nil, http.StatusInternalServerError, err)
			return
		}
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
	// Mark if sources are enabled or not
	sourcesEnabled, err := storage.GetUserSourcesEnabled(user.Name)
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, errors.Wrap(err, "failed to retrieve sources enabled"))
		return
	}
	sourceEnabledMark := map[uint]bool{}
	for _, v := range sourcesEnabled {
		sourceEnabledMark[v.ID] = true
	}
	for i, v := range sources {
		sources[i].Enabled = sourceEnabledMark[v.ID]

		// let's also check if crawling is in progress
		crawler := genCrawlerName(user.Name, sources[i].Link)
		scrape, err := storage.GetLastScrape(crawler)
		// litter.Dump(crawler)
		// litter.Dump(scrape)
		if err == nil {
			// Elapsed = 0 because it hasn't been set when
			// crawler finished crawling the link it had
			sources[i].Crawling = scrape.Elapsed == 0
		} else {
			sources[i].Crawling = false
		}
	}
	httpJSON(w, sources, http.StatusOK, nil)
}
