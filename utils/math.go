package utils

// Max returns the Max of its arguments
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the Min of its arguments
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
