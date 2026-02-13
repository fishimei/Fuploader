package baijiahao

import "unicode/utf8"

func TruncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

func StringRuneLen(s string) int {
	return utf8.RuneCountInString(s)
}
