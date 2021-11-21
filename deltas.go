package main

import (
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/thecsw/katya/log"
	"github.com/thecsw/katya/storage"
)

const (
	// globalDeltaCacheKey is a small map key used in delta cache for globals
	globalDeltaCacheKey = "g"
	// deltaUpdateInterval is how frequently should we update deltas
	deltaUpdateInterval = time.Minute
)

var (
	// globalNumWordsDelta caches yet-to-be-updated deltas in global word count
	globalNumWordsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	// globalNumSentencesDelta caches yet-to-be-updated deltas in global sentences count
	globalNumSentencesDelta = cache.New(cache.NoExpiration, cache.NoExpiration)

	// sourcesNumWordsDelta caches yet-to-be-updated deltas in sources' word count
	sourcesNumWordsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	// sourcesNumSentencesDelta caches yet-to-be-updated deltas in sources' sentences count
	sourcesNumSentencesDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
)

// updateGlobalWordSentencesDeltas updates global deltas of num_words and num_sentences
func updateGlobalWordSentencesDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(deltaUpdateInterval)
		// whether we should print an update message at the end or not
		actuallyUpdated := false
		//l("Starting updating the global words/sentences count")
		// Update the word count
		wordDelta, _ := globalNumWordsDelta.Get(globalDeltaCacheKey)
		if wordDelta.(uint) != 0 {
			if err := storage.UpdateGlobalWordNum(wordDelta.(uint)); err != nil {
				log.Error("failed updating global word count", err, log.Params{})
				continue
			}
			actuallyUpdated = true
		}
		// Update the sentences count
		sentDelta, _ := globalNumSentencesDelta.Get(globalDeltaCacheKey)
		if sentDelta.(uint) != 0 {
			if err := storage.UpdateGlobalSentNum(sentDelta.(uint)); err != nil {
				log.Error("failed updating global sentences count", err, log.Params{})
				continue
			}
			actuallyUpdated = true
		}
		// Drain the cache
		globalNumWordsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		globalNumSentencesDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		// Log the info
		if actuallyUpdated {
			log.Info("Successfully updated the global words/sentences count")
		}
	}
}

// updateGlobalWordSentencesDeltas updates sources' deltas of num_words and num_sentences
func updateSourcesWordSentencesDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(deltaUpdateInterval)
		// whether we should print an update message at the end or not
		actuallyUpdated := false
		//l("Starting to update sources' words/sentences count")
		// Update the word count
		wordItems := sourcesNumWordsDelta.Items()
		for k, v := range wordItems {
			delta := v.Object.(uint)
			if delta == 0 {
				continue
			}
			if err := storage.UpdateSourceWordNum(k, delta); err != nil {
				log.Error("failed updating source word count", err, log.Params{
					"source": k,
				})
				continue
			}
			actuallyUpdated = true
			sourcesNumWordsDelta.Set(k, uint(0), cache.NoExpiration)

		}
		// Update the sentences count
		sentItems := sourcesNumSentencesDelta.Items()
		for k, v := range sentItems {
			delta := v.Object.(uint)
			if delta == 0 {
				continue
			}
			if err := storage.UpdateSourceSentNum(k, delta); err != nil {
				log.Error("failed updating source sentences count", err, log.Params{
					"source": k,
				})
				continue
			}
			actuallyUpdated = true
			sourcesNumSentencesDelta.Set(k, uint(0), cache.NoExpiration)
		}
		// Log the info
		if actuallyUpdated {
			log.Info("Successfully update sources' words/sentences count")
		}
	}
}
