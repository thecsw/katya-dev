package main

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name string `json:"name"`
}

type Source struct {
	gorm.Model
	UserID uint   `json:"user_id"`
	Link   string `json:"link"`
}

type Text struct {
	gorm.Model
	SourceID  uint   `json:"source_id"`
	NameCrawl string `json:"name"`
	URL       string `json:"url"`
	IP        string `json:"ip"`
	Status    int    `json:"status"`
	Text      string `json:"text"`
}
