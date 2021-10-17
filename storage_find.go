package main

import "strings"

var (
	// findByPart maps a text type to the find function of its type
	findByPart = map[string]func(uint, string, int, int, bool) ([]Text, error){
		"text":   findTextsByUserID,
		"shapes": findShapesByUserID,
		"tags":   findTagsByUserID,
		"nomins": findNominativesByUserID,
	}
)

// findTextsByUserID runs a DB search against the text part of texts
func findTextsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.text", userID, query, limit, offset, caseSensitive)
}

// findShapesByUserID runs a DB search against the shapes part of texts
func findShapesByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.shapes", userID, query, limit, offset, caseSensitive)
}

// findTagsByUserID runs a DB search against the tags part of texts
func findTagsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.tags", userID, query, limit, offset, caseSensitive)
}

// findNominativesByUserID runs a DB search against the nominatives part of texts
func findNominativesByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.nominatives", userID, query, limit, offset, caseSensitive)
}

// findTextsPartsByUserID is the lower-level-true-SQL fundamental function to seacrh parts of texts
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
