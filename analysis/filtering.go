package analysis

import (
	"sort"
	"strconv"
	"unicode"
)

type PairSimple struct {
	Key   string
	Value uint
}

type PairListSimple []PairSimple

func (p PairListSimple) Len() int           { return len(p) }
func (p PairListSimple) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairListSimple) Less(i, j int) bool { return p[i].Value > p[j].Value }

type Pair struct {
	Key   string
	Value Relation
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool { return p[i].Value.Occured > p[j].Value.Occured }

func FilterStopwords(results map[string]*Relation, stoplist map[string]bool) PairList {
	//toWrite := make([][][]string, 0, len(results)+1)
	p := make(PairList, 0, len(results))
	i := 0
	for k, v := range results {
		if _, isStopword := stoplist[k]; isStopword {
			continue
		}
		if unicodeIsThis(k, unicode.IsPunct) || unicodeIsThis(k, unicode.IsSymbol) {
			continue
		}
		p = append(p, Pair{k, *v})
		i++
	}
	sort.Sort(p)
	// for _, v := range p {
	// 	//toWrite = append(toWrite, []string{v.Key, strconv.FormatUint(uint64(v.Value), 10)})
	// }
	return p
}

func FilterStopwordsSimple(results map[string]uint, stoplist map[string]bool) [][]string {
	toWrite := make([][]string, 0, len(results)+1)
	p := make(PairListSimple, 0, len(results))
	i := 0
	for k, v := range results {
		if _, isStopword := stoplist[k]; isStopword {
			continue
		}
		if unicodeIsThis(k, unicode.IsPunct) || unicodeIsThis(k, unicode.IsSymbol) {
			continue
		}
		p = append(p, PairSimple{k, v})
		i++
	}
	sort.Sort(p)
	for _, v := range p {
		toWrite = append(toWrite, []string{v.Key, strconv.FormatUint(uint64(v.Value), 10)})
	}
	return toWrite
}
