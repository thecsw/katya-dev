package utils

import "strings"

// StringsIndexMultiple finds all indices where a substring occurs
// in the given string, whether it's a case sensitive search
// or not can be adjusted by the user
func StringsIndexMultiple(s, subs string, caseSensitive bool) []int {
	if !caseSensitive {
		s = strings.ToLower(s)
		subs = strings.ToLower(subs)
	}
	res := make([]int, 0)
	start := 0
	for {
		index := strings.Index(s[start:], subs)
		if index == -1 {
			break
		}
		res = append(res, index+start)
		start = start + index + len(subs)
		if start >= len(s) {
			break
		}
	}
	return res
}

// ReverseString from https://groups.google.com/g/golang-nuts/c/oPuBaYJ17t4
func ReverseString(what string) string {
	// Get Unicode code points.
	n := 0
	rune := make([]rune, len(what))
	for _, r := range what {
		rune[n] = r
		n++
	}
	rune = rune[0:n]
	// Reverse
	for i := 0; i < n/2; i++ {
		rune[i], rune[n-1-i] = rune[n-1-i], rune[i]
	}
	// Convert back to UTF-8.
	return string(rune)
}

// FindTokenIndex maps a found index to a tokenized slice index
func FindTokenIndex(tokens []string, index int) int {
	currentSum := 0
	for i, v := range tokens {
		if currentSum > index {
			return i - 1
		}
		currentSum += len(v) + 1
	}
	return -1
}
