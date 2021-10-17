package main

import "github.com/patrickmn/go-cache"

func createCrawler(name, user, source string) error {
	userObj, err := getUser(user, false)
	if err != nil {
		return err
	}
	sourceObj, err := getSource(source, false)
	if err != nil {
		return err
	}
	err = DB.Create(&Crawler{
		Name:     name,
		SourceID: sourceObj.ID,
		UserID:   userObj.ID,
	}).Error
	if err != nil {
		return err
	}
	lf("Successfully created a new crawler", params{
		"name":   name,
		"user":   user,
		"source": source,
	})
	return nil
}

func getCrawler(name string, fill bool) (*Crawler, error) {
	crawlerObj := &Crawler{}
	if ID, found := crawlerToID.Get(name); found {
		// Don't ping DB to fill the object
		if !fill {
			crawlerObj.ID = ID.(uint)
			return crawlerObj, nil
		}
		return crawlerObj, DB.First(crawlerObj, ID.(uint)).Error
	}
	err := DB.First(crawlerObj, "name = ?", name).Error
	if err != nil {
		return nil, err
	}
	crawlerToID.Set(name, crawlerObj.ID, cache.NoExpiration)
	return crawlerObj, nil
}

func isCrawler(name string) (bool, error) {
	if _, found := crawlerToID.Get(name); found {
		return true, nil
	}
	count := int64(0)
	err := DB.First(&Crawler{}, "name = ?", name).Count(&count).Error
	return count != 0, err
}
