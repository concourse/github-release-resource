package resource

import (
	"errors"
	"regexp"

	"github.com/cppforlife/go-semi-semantic/version"
	"github.com/google/go-github/github"
)

func FilterByVersion(releases []*github.RepositoryRelease, predicateString string) ([]*github.RepositoryRelease, error) {
	if predicateString == "" {
		return releases, nil
	}

	predicate, err := ParsePredicate(predicateString)
	if err != nil {
		return []*github.RepositoryRelease{}, err
	}

	var filteredReleases []*github.RepositoryRelease

	for _, release := range releases {
		if predicate.Apply(*release.TagName) {
			filteredReleases = append(filteredReleases, release)
		}
	}

	return filteredReleases, err
}

type VersionPredicate struct {
	Condition string
	Version   string
}

func ParsePredicate(filter string) (VersionPredicate, error) {
	re, err := regexp.Compile(`(<)\s*(.*)`)
	if err != nil {
		return VersionPredicate{}, err
	}

	matches := re.FindAllStringSubmatch(filter, -1)
	if len(matches) != 1 || len(matches[0]) != 3 {
		return VersionPredicate{}, errors.New("Invalid version filter")
	}

	return VersionPredicate{Condition: matches[0][1], Version: matches[0][2]}, nil
}

func (p VersionPredicate) Apply(version string) bool {
	return lessThan(version, p.Version)
}

func lessThan(v1, v2 string) bool {
	first, err := version.NewVersionFromString(determineVersionFromTag(v1))
	if err != nil {
		return true
	}

	second, err := version.NewVersionFromString(determineVersionFromTag(v2))
	if err != nil {
		return false
	}

	return first.IsLt(second)
}
