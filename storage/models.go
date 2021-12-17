package storage

import "gorm.io/gorm"

// User struct defines the user of the system, a user can have
// multiple sources associated with a user.
type User struct {
	gorm.Model `json:"-"`

	// Name is the username
	Name string `json:"name" gorm:"unique"`
	// Password is the user's password (sha256 hashed)
	Password string `json:"-"`

	// User can have multiple sources, a source can have multiple users
	Sources []*Source `gorm:"many2many:user_sources;" json:"-"`

	// SourcesEnabled maps to sources that a user has, but only enabled ones
	SourcesEnabled []*Source `gorm:"many2many:user_sources_enabled;" json:"-"`
}

// Source struct defines a source website that is given by a user
// for us to crawl.
type Source struct {
	gorm.Model `json:"-"`

	// Link is the starting link associated with the source
	Link string `json:"link" gorm:"unique"`
	// Label is the user-given label for the source
	Label string `json:"label"`
	// NumWords is the number of words for the whole source
	NumWords uint `json:"num_words"`
	// NumSentences is the number of sentences for the whole source
	NumSentences uint `json:"num_sentences"`

	// Each source has multiple texts and each text can be linked
	// to from multiple different source (overlapping links)
	Texts []*Text `gorm:"many2many:source_texts;" json:"-"`
	// Each source can be connected to multiple users and each
	// user can have multiple sources on their account
	Users []*User `gorm:"many2many:user_sources;" json:"-"`
	// UsersEnabled maps to sources that a user has, but only enabled ones
	UsersEnabled []*User `gorm:"many2many:user_sources_enabled;" json:"-"`

	// JSON-specific exports
	// Enabled flags if the source is enabled when exported
	Enabled  bool `gorm:"-" json:"enabled"`
	Crawling bool `gorm:"-" json:"crawling"`
}

// Crawler struct defines the crawlers that we have, with the starting
// link they're using and the user that created this crawler.
type Crawler struct {
	gorm.Model `json:"-"`

	// Name is the name of the crawler
	Name string `json:"name" gorm:"unique"`
	// SourceID is the ID of the source its crawling
	SourceID uint `json:"source_id"`
	// UserID is the ID of the user that own the crawler
	UserID uint `json:"user_id"`
}

// Scrape struct stores all the crawlers runs, only associated with a
// single crawler, it stores the time in UTC unix timestamps.
type Scrape struct {
	gorm.Model `json:"-"`

	// CrawlerID is the crawler that scrape refers to
	CrawlerID uint `json:"crawler_id"`
	// Start is the UNIX UTC timestamp of when scrape started
	Start uint `json:"start"`
	// End is the UNIX UTC timestamp of when scrape finished
	End uint `json:"end"`
	// Elapsed is simply "end - start"
	Elapsed uint `json:"elapsed"`
}

// Global struct gives us info about the whole corpus.
type Global struct {
	gorm.Model `json:"-"`

	// NumWords is the number of words across ALL texts
	NumWords uint `json:"num_words"`
	// NumSentences is the number of sentences across ALL texts
	NumSentences uint `json:"num_sentences"`
}

// Text struct actually stores the data that we scraped, it's back-linked
// to sources, such that a source can have multiple texts and a text can have
// multiple sources (overlapping URLs found during crawling).
type Text struct {
	gorm.Model `json:"-"`

	// URL is the web resources where we extract text from
	URL string `json:"url" gorm:"unique"`
	// IP is the ip address that we pulled the text from
	IP string `json:"ip"`
	// Status is the HTTP return status code we got from URL
	Status uint `json:"status"`
	// Original is the cleaned text we pulled from the page
	Original string `json:"original"`
	// Text is the tokenized SpaCy original text (spaces around punct)
	Text string `json:"text"`
	// Shapes is the tokenized shapes from SpaCy
	Shapes string `json:"shapes"`
	// Tags is the tokenized text from SpaCy
	Tags string `json:"tags"`
	// Lemmas is the tokenized text of nominatives from SpaCy
	Lemmas string `json:"lemmas"`
	// Title is the title of the HTML webpage (extracted)
	Title string `json:"title"`
	// NumWords is the number of words (no punct) of the Text
	NumWords uint `json:"num_words"`
	// NumWords is the number of sentences of the Text
	NumSentences uint `json:"num_sentences"`

	// Text can be associated with multiple sources and a source
	// can be associated with many texts
	Sources []*Source `gorm:"many2many:source_texts;" json:"-"`
}
