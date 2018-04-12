package resource

import (
	"regexp"
	"strconv"

	"github.com/google/go-github/github"
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

func versionFromRelease(release *github.RepositoryRelease) Version {
	if *release.Draft {
		return Version{ID: strconv.Itoa(*release.ID)}
	} else {
		return Version{Tag: *release.TagName}
	}
}
