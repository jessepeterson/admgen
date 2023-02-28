package admgen

import "strings"

func strip(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		b := s[i]
		if ('a' <= b && b <= 'z') ||
			('A' <= b && b <= 'Z') ||
			('0' <= b && b <= '9') {
			result.WriteByte(b)
		}
	}
	return result.String()
}

func normalizeFieldName(s string) string {
	s = strings.ToUpper(s[0:1]) + s[1:]
	return strip(s)
}
