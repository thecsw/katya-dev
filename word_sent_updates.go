package main

import (
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	globalDeltaCacheKey = "g"
	deltaUpdateInterval = time.Minute
)

var (
	globalNumWordsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	globalNumSentsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)

	sourcesNumWordsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
	sourcesNumSentsDelta = cache.New(cache.NoExpiration, cache.NoExpiration)
)

// updateGlobalWordSentsDeltas updates global deltas of num_words and num_sents
func updateGlobalWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(deltaUpdateInterval)
		actuallyUpdated := false
		//l("Starting updating the global words/sents count")
		// Update the word count
		wordDelta, _ := globalNumWordsDelta.Get(globalDeltaCacheKey)
		if wordDelta.(uint) != 0 {
			if err := updateGlobalWordNum(wordDelta.(uint)); err != nil {
				lerr("failed updating global word count", err, params{})
				continue
			}
			actuallyUpdated = true
		}
		// Update the sentences count
		sentDelta, _ := globalNumSentsDelta.Get(globalDeltaCacheKey)
		if sentDelta.(uint) != 0 {
			if err := updateGlobalSentNum(sentDelta.(uint)); err != nil {
				lerr("failed updating global word count", err, params{})
				continue
			}
			actuallyUpdated = true
		}
		// Drain the cache
		globalNumWordsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		globalNumSentsDelta.Set(globalDeltaCacheKey, uint(0), cache.NoExpiration)
		// Log the info
		if actuallyUpdated {
			l("Successfully updated the global words/sents count")
		}
	}
}

// updateGlobalWordSentsDeltas updates sources' deltas of num_words and num_sents
func updateSourcesWordSentsDeltas() {
	for {
		// Sleep for a minute
		time.Sleep(deltaUpdateInterval)
		actuallyUpdated := false
		//l("Starting to update sources' words/sents count")
		// Update the word count
		wordItems := sourcesNumWordsDelta.Items()
		for k, v := range wordItems {
			delta := v.Object.(uint)
			if delta == 0 {
				continue
			}
			if err := updateSourceWordNum(k, delta); err != nil {
				lerr("failed updating source word count", err, params{
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
			if err := updateSourceSentNum(k, delta); err != nil {
				lerr("failed updating source sent count", err, params{
					"source": k,
				})
				continue
			}
			actuallyUpdated = true
			sourcesNumSentsDelta.Set(k, uint(0), cache.NoExpiration)
		}
		// Log the info
		if actuallyUpdated {
			l("Successfully update sources' words/sents count")
		}
	}
}
