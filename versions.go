package resource

import (
	"regexp"
	"strconv"

	"github.com/zachgersh/go-github/github"
)

// determineVersionFromTag converts git tags v1.2.3 into semver 1.2.3 values
func determineVersionFromTag(tag string) string {
	re := regexp.MustCompile("v?([^v].*)")
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 0 {
		return matches[1]
	} else {
		return ""
	}
}

func versionFromRelease(release *github.RepositoryRelease) Version {
	if *release.Draft {
		return Version{ID: strconv.Itoa(*release.ID)}
	} else {
		return Version{Tag: *release.TagName}
	}
}
