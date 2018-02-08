package resource

import (
	"errors"
	"regexp"

	"github.com/cppforlife/go-semi-semantic/version"
)

func FilterVersions(versions []Version, predicateString string) ([]Version, error) {
	if predicateString == "" {
		return versions, nil
	}

	predicate, err := ParsePredicate(predicateString)
	if err != nil {
		return []Version{}, err
	}

	var filteredVersions []Version

	for _, version := range versions {
		if predicate.Apply(version) {
			filteredVersions = append(filteredVersions, version)
		}
	}

	return filteredVersions, err
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

func (p VersionPredicate) Apply(version Version) bool {
	return lessThan(version, Version{Tag: p.Version})
}

func lessThan(v1, v2 Version) bool {
	first, err := version.NewVersionFromString(determineVersionFromTag(v1.Tag))
	if err != nil {
		return true
	}

	second, err := version.NewVersionFromString(determineVersionFromTag(v2.Tag))
	if err != nil {
		return false
	}

	return first.IsLt(second)
}
