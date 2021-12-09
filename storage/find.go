package storage

import "strings"

var (
	// MapPartToFindFunction maps a text type to the find function of its type
	MapPartToFindFunction = map[string]func(uint, string, int, int, bool) ([]Text, error){
		"text":   FindTextsByUserID,
		"shapes": FindShapesByUserID,
		"tags":   FindTagsByUserID,
		"lemmas": FindLemmasByUserID,
	}
)

// FindTextsByUserID runs a DB search against the text part of texts
func FindTextsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.text", userID, query, limit, offset, caseSensitive)
}

// FindShapesByUserID runs a DB search against the shapes part of texts
func FindShapesByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.shapes", userID, query, limit, offset, caseSensitive)
}

// FindTagsByUserID runs a DB search against the tags part of texts
func FindTagsByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.tags", userID, query, limit, offset, caseSensitive)
}

// FindLemmasByUserID runs a DB search against the nominatives part of texts
func FindLemmasByUserID(userID uint,
	query string,
	limit int,
	offset int,
	caseSensitive bool,
) ([]Text, error) {
	return findTextsPartsByUserID("texts.lemmas", userID, query, limit, offset, caseSensitive)
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
		Joins("INNER JOIN user_sources_enabled on user_sources.source_id = user_sources_enabled.source_id").
		Joins("INNER JOIN users on user_sources_enabled.user_id = users.id AND users.id = ?", userID).
		Where(sqlWhere, sqlMatch).
		Limit(limit).
		Offset(offset).
		Find(&texts).
		Error
	return texts, err
}
