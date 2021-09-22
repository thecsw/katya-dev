package main

import "gorm.io/gorm"

// User struct defines the user of the system, a user can have
// multiple sources associated with a user.
type User struct {
	gorm.Model

	Name     string `json:"name" gorm:"unique"`
	Password string `json:"-"`

	Sources []*Source `gorm:"many2many:user_sources;" json:"-"`
}

// Source struct defines a source website that is given by a user
// for us to crawl.
type Source struct {
	gorm.Model

	Link         string `json:"link" gorm:"unique"`
	NumWords     uint   `json:"num_words"`
	NumSentences uint   `json:"num_sents"`

	Texts []*Text `gorm:"many2many:source_texts;" json:"-"`
	Users []*User `gorm:"many2many:user_sources;" json:"-"`
}

// Crawler struct defines the crawlers that we have, with the starting
// link they're using and the user that created this crawler.
type Crawler struct {
	gorm.Model

	Name     string `json:"name" gorm:"unique"`
	SourceID uint   `json:"source_id"`
	UserID   uint   `json:"user_id"`
}

// Scrape struct stores all the crawlers runs, only associated with a
// single crawler, it stores the time in UTC unix timestamps.
type Scrape struct {
	gorm.Model

	CrawlerID uint `json:"crawler_id"`
	Start     uint `json:"start"`
	Elapsed   uint `json:"elapsed"`
	End       uint `json:"end"`
}

// Global struct gives us info about the whole corpus.
type Global struct {
	gorm.Model

	NumWords     uint `json:"num_words"`
	NumSentences uint `json:"num_sents"`
}

// Text struct actually stores the data that we scraped, it's back-linked
// to sources, such that a source can have multiple texts and a text can have
// multiple sources (overlapping URLs found during crawling).
type Text struct {
	gorm.Model

	URL      string `json:"url" gorm:"unique"`
	IP       string `json:"ip"`
	Status   uint   `json:"status"`
	Text     string `json:"text"`
	Title    string `json:"title"`
	NumWords uint   `json:"num_words"`
	NumSents uint   `json:"num_sents"`

	Sources []*Source `gorm:"many2many:source_texts;" json:"-"`
}
