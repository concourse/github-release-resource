package resource

import (
	"regexp"
	"strconv"
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
	if len(matches) > 0 {
		return matches[len(matches)-1]
	}
	return ""
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
