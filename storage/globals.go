package storage

// CreateGlobal creates a global instance
func CreateGlobal() error {
	return DB.Create(&Global{NumWords: uint(0)}).Error
}

// DoesGlobalExist checks whether a global instance exists
func DoesGlobalExist() bool {
	count := int64(0)
	DB.First(&Global{}).Count(&count)
	return count != 0
}

// GetNumOfSources returns the global number of sources
func GetNumOfSources() (uint, error) {
	count := uint(0)
	return count, DB.
		Raw("SELECT count(1) FROM sources").
		Scan(&count).
		Error
}

// UpdateGlobalWordNum returns the global number of words
func UpdateGlobalWordNum(numWords uint) error {
	return DB.Exec(
		"UPDATE globals SET num_words = num_words + ? where id = 1",
		numWords).
		Error
}

// UpdateGlobalSentNum returns the global number of sentences
func UpdateGlobalSentNum(numSents uint) error {
	return DB.Exec(
		"UPDATE globals SET num_sentences = num_sentences + ? where id = 1",
		numSents).
		Error
}
