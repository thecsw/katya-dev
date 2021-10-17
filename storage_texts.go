package main

import (
	"errors"
	"strings"

	"gorm.io/gorm/clause"
)

func createText(
	source string,
	url string,
	ip string,
	status uint,
	original string,
	text string,
	shapes string,
	tags string,
	nomins string,
	title string,
	numWords uint,
	numSents uint,
) error {
	sourceObj, err := getSource(source, false)
	if err != nil {
		return err
	}
	toAdd := &Text{
		URL:         url,
		IP:          ip,
		Status:      status,
		Original:    original,
		Text:        text,
		Shapes:      shapes,
		Tags:        tags,
		Nominatives: nomins,
		Title:       title,
		NumWords:    numWords,
		NumSents:    numSents,
		Sources:     []*Source{},
	}
	err = DB.
		Model(sourceObj).
		Clauses(clause.OnConflict{
			DoNothing: true,
			UpdateAll: true,
		}).
		Association("Texts").
		Append(toAdd)
	if err != nil {
		if strings.Contains(err.Error(), "SQLSTATE 23503") {
			lf("Text link already exists, not replacing.", params{
				"url":   url,
				"title": title,
			})
			return errors.New("already exists")
		}
		return err
	}
	lf("Successfully created a new text", params{
		"url":       url,
		"title":     title,
		"ip":        ip,
		"num_words": numWords,
		"num_sents": numSents,
	})
	return nil
}

func findTexts(
	user string,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	texts := make([]Text, 0, limit)
	sqlWhere := "texts.text LIKE ?"
	sqlMatch := "%" + query + "%"
	if !caseSensitive {
		sqlWhere = "lower(texts.text) LIKE ?"
		sqlMatch = "%" + strings.ToLower(query) + "%"
	}
	err := DB.Model(texts).
		Joins("JOIN source_texts on texts.id = source_texts.text_id").
		Joins("JOIN sources on sources.id = source_texts.source_id").
		Joins("JOIN user_sources on sources.id = user_sources.source_id").
		Joins("JOIN users on user_sources.user_id = users.id AND users.name = ?", user).
		Where(sqlWhere, sqlMatch).
		Limit(limit).
		Offset(offset).
		Find(&texts).
		Error
	return texts, err
}

var (
	findByPart = map[string]func(uint, string, int, int, bool) ([]Text, error){
		"text":   findTextsByUserID,
		"shapes": findShapesByUserID,
		"tags":   findTagsByUserID,
		"nomins": findNominativesByUserID,
	}
)

func findTextsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.text", userID, query, limit, offset, caseSensitive)
}

func findShapesByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.shapes", userID, query, limit, offset, caseSensitive)
}

func findTagsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.tags", userID, query, limit, offset, caseSensitive)
}

func findNominativesByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.nominatives", userID, query, limit, offset, caseSensitive)
}

func findTextsPartsByUserID(
	part string,
	userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	texts := make([]Text, 0, limit)
	sqlWhere := part + " LIKE ?"
	sqlMatch := "%" + query + "%"
	if !caseSensitive {
		sqlWhere = "lower(" + part + ") LIKE ?"
		sqlMatch = "%" + strings.ToLower(query) + "%"
	}
	err := DB.Model(texts).
		Joins("INNER JOIN source_texts on texts.id = source_texts.text_id").
		Joins("INNER JOIN sources on sources.id = source_texts.source_id").
		Joins("INNER JOIN user_sources on sources.id = user_sources.source_id").
		Joins("INNER JOIN users on user_sources.user_id = users.id AND users.id = ?", userID).
		Where(sqlWhere, sqlMatch).
		Limit(limit).
		Offset(offset).
		Find(&texts).
		Error
	return texts, err
}
