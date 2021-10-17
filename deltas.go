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
	// globalNumSentsDelta caches yet-to-be-updated deltas in global sentences count
	globalNumSentsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)

	// sourcesNumWordsDelta caches yet-to-be-updated deltas in sources' word count
	sourcesNumWordsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	// sourcesNumSentsDelta caches yet-to-be-updated deltas in sources' sentences count
	sourcesNumSentsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
)

// updateGlobalWordSentsDeltas updates global deltas of num_words and num_sents
func updateGlobalWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(deltaUpdateInterval)
		// whether we should print an update message at the end or not
		actuallyUpdated := false
		//l("Starting updating the global words/sents count")
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
		sentDelta, _ := globalNumSentsDelta.Get(globalDeltaCacheKey)
		if sentDelta.(uint) != 0 {
			if err := storage.UpdateGlobalSentNum(sentDelta.(uint)); err != nil {
				log.Error("failed updating global word count", err, log.Params{})
				continue
			}
			actuallyUpdated = true
		}
		// Drain the cache
		globalNumWordsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		globalNumSentsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		// Log the info
		if actuallyUpdated {
			log.Info("Successfully updated the global words/sents count")
		}
	}
}

// updateGlobalWordSentsDeltas updates sources' deltas of num_words and num_sents
func updateSourcesWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(deltaUpdateInterval)
		// whether we should print an update message at the end or not
		actuallyUpdated := false
		//l("Starting to update sources' words/sents count")
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
		// Update the sents count
		sentItems := sourcesNumSentsDelta.Items()
		for k, v := range sentItems {
			delta := v.Object.(uint)
			if delta == 0 {
				continue
			}
			if err := storage.UpdateSourceSentNum(k, delta); err != nil {
				log.Error("failed updating source sent count", err, log.Params{
					"source": k,
				})
				continue
			}
			actuallyUpdated = true
			sourcesNumSentsDelta.Set(k, uint(0), cache.NoExpiration)
		}
		// Log the info
		if actuallyUpdated {
			log.Info("Successfully update sources' words/sents count")
		}
	}
}
