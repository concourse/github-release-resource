package resource

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"
)

var defaultTagFilter = "^v?([^v].*)"

type versionParser struct {
	re *regexp.Regexp
}

func newVersionParser(filter string) (versionParser, error) {
	if filter == "" {
		filter = defaultTagFilter
	}
	re, err := regexp.Compile(filter)
	if err != nil {
		return versionParser{}, err
	}
	return versionParser{re: re}, nil
}

func (vp *versionParser) parse(tag string) string {
	matches := vp.re.FindStringSubmatch(tag)

	// If regex doesn't match at all, return empty
	if len(matches) == 0 {
		return ""
	}

	// If regex has a capture group, try to use it
	if len(matches) > 1 {
		captured := matches[1]
		// Only use capture group if it looks like a reasonable version
		// (at least 3 chars or contains dots for semver)
		if len(captured) >= 3 || strings.Contains(captured, ".") {
			return captured
		}
	}

	// Fallback: use full match and strip common version prefixes
	version := matches[0]
	prefixes := []string{
		"v",
		"version-",
		"release-",
		"rel-",
		"r",
		"v-",
		"@",
		"stable-",
		"final-",
		"prod-",
		"production-",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(version, prefix) {
			version = strings.TrimPrefix(version, prefix)
			break
		}
	}
	return version
}

func getTimestamp(release *github.RepositoryRelease) time.Time {
	if release.PublishedAt != nil {
		return release.PublishedAt.Time
	} else if release.CreatedAt != nil {
		return release.CreatedAt.Time
	} else {
		return time.Time{}
	}
}

func versionFromRelease(release *github.RepositoryRelease) Version {
	v := Version{
		ID:        strconv.FormatInt(*release.ID, 10),
		Timestamp: getTimestamp(release),
	}
	if release.TagName != nil {
		v.Tag = *release.TagName
	}
	return v
}
