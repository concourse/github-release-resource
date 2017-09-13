package resource

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
)

// determineVersionFromTag converts git tags v1.2.3 into semver 1.2.3 values
func determineVersionFromTagBollocks(tag string, source Source) string {
	var re *regexp.Regexp
	if source.TagFilterRegex == "" {
		re = regexp.MustCompile(`^v?(?:\d+\.)?(?:\d+\.)?(?:\*|\d+.*)$`)
		tag = strings.TrimPrefix(tag, "v")
	} else {
		re = regexp.MustCompile(source.TagFilterRegex)
	}

	if re.MatchString(tag) {
		return tag
	}
	return ""
}

// determineVersionFromTag converts git tags v1.2.3 into semver 1.2.3 values
func determineVersionFromTag(tag string) string {
	re := regexp.MustCompile(`^v?(\d+\.)?(\d+\.)?(\*|\d+.*)$`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 0 {
		return strings.Join(matches[1:], "")
	}
	return ""
}

func versionFromRelease(release *github.RepositoryRelease) Version {
	if *release.Draft {
		return Version{ID: strconv.Itoa(*release.ID)}
	}
	return Version{Tag: *release.TagName}
}

func tagMatchesFilter(source Source, tagName string) bool {
	re := regexp.MustCompile(source.TagFilterRegex)
	return re.MatchString(tagName)
}
