package main

func createGlobal() error {
	return DB.Create(&Global{NumWords: uint(0)}).Error
}

func doesGlobalExist() bool {
	count := int64(0)
	DB.First(&Global{}).Count(&count)
	return count != 0
}

func getNumOfSources() (uint, error) {
	count := uint(0)
	return count, DB.
		Raw("SELECT count(1) FROM sources").
		Scan(&count).
		Error
}

func updateGlobalWordNum(numWords uint) error {
	return DB.Exec(
		"UPDATE globals SET num_words = num_words + ? where id = 1",
		numWords).
		Error
}

func updateGlobalSentNum(numSents uint) error {
	return DB.Exec(
		"UPDATE globals SET num_sentences = num_sentences + ? where id = 1",
		numSents).
		Error
}
