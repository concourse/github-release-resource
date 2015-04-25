package resource

import "regexp"

// determineVersionFromTag converts git tags v1.2.3 into semver 1.2.3 values
func determineVersionFromTag(tag string) string {
	re := regexp.MustCompile("v?([^v].*)")
	matches := re.FindStringSubmatch(tag)
	return matches[1]
}
