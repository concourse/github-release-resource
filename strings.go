package resource

import "unicode"

func dropLeadingAlpha(s string) string {
	for i, r := range s {
		if !unicode.IsLetter(r) {
			return s[i:]
		}
	}

	return ""
}
