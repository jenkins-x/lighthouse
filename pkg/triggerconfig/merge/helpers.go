package merge

// StringArrayIndex returns the index in the slice which equals the given value
func StringArrayIndex(array []string, value string) int {
	for i, v := range array {
		if v == value {
			return i
		}
	}
	return -1
}

// RemoveStringArrayAtIndex removes an item at a given index
func RemoveStringArrayAtIndex(s []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}
