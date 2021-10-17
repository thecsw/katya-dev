package main

// createGlobal creates a global instance
func createGlobal() error {
	return DB.Create(&Global{NumWords: uint(0)}).Error
}

// doesGlobalExist checks whether a global instance exists
func doesGlobalExist() bool {
	count := int64(0)
	DB.First(&Global{}).Count(&count)
	return count != 0
}

// getNumOfSources returns the global number of sources
func getNumOfSources() (uint, error) {
	count := uint(0)
	return count, DB.
		Raw("SELECT count(1) FROM sources").
		Scan(&count).
		Error
}

// updateGlobalWordNum returns the global number of words
func updateGlobalWordNum(numWords uint) error {
	return DB.Exec(
		"UPDATE globals SET num_words = num_words + ? where id = 1",
		numWords).
		Error
}

// updateGlobalSentNum returns the global number of sentences
func updateGlobalSentNum(numSents uint) error {
	return DB.Exec(
		"UPDATE globals SET num_sentences = num_sentences + ? where id = 1",
		numSents).
		Error
}
