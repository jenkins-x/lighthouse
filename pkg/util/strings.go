package util

// StringArrayIndex returns the index in the slice which equals the given value
func StringArrayIndex(array []string, value string) int {
	for i, v := range array {
		if v == value {
			return i
		}
	}
	return -1
}
