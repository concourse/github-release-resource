package resource

import (
	"sort"
	"strconv"

	"github.com/google/go-github/github"

	"github.com/cppforlife/go-semi-semantic/version"
)

type CheckCommand struct {
	github GitHub
}

func NewCheckCommand(github GitHub) *CheckCommand {
	return &CheckCommand{
		github: github,
	}
}

func (c *CheckCommand) Run(request CheckRequest) ([]Version, error) {
	releases, err := c.github.ListReleases()
	if err != nil {
		return []Version{}, err
	}

	if len(releases) == 0 {
		return []Version{}, nil
	}

	var filteredReleases []*github.RepositoryRelease

	versionParser, err := newVersionParser(request.Source.TagFilter)
	if err != nil {
		return []Version{}, err
	}

	for _, release := range releases {
		if request.Source.Drafts != *release.Draft {
			continue
		}

		// Should we skip this release
		//   a- prerelease condition dont match our source config
		//   b- release condition match  prerealse in github since github has true/false to describe release/prerelase
		if request.Source.PreRelease != *release.Prerelease && request.Source.Release == *release.Prerelease {
			continue
		}

		if release.TagName == nil {
			continue
		}
		if _, err := version.NewVersionFromString(versionParser.parse(*release.TagName)); err != nil {
			continue
		}

		filteredReleases = append(filteredReleases, release)
	}

	sort.Slice(filteredReleases, func(i, j int) bool {
		first, err := version.NewVersionFromString(versionParser.parse(*filteredReleases[i].TagName))
		if err != nil {
			return true
		}

		second, err := version.NewVersionFromString(versionParser.parse(*filteredReleases[j].TagName))
		if err != nil {
			return false
		}

		return first.IsLt(second)
	})

	if len(filteredReleases) == 0 {
		return []Version{}, nil
	}
	latestRelease := filteredReleases[len(filteredReleases)-1]

	if (request.Version == Version{}) {
		return []Version{
			versionFromRelease(latestRelease),
		}, nil
	}

	if *latestRelease.TagName == request.Version.Tag {
		return []Version{}, nil
	}

	upToLatest := false
	reversedVersions := []Version{}

	for _, release := range filteredReleases {
		if !upToLatest {
			if *release.Draft || *release.Prerelease {
				id := *release.ID
				upToLatest = request.Version.ID == strconv.Itoa(id)
			} else {
				version := *release.TagName
				upToLatest = request.Version.Tag == version
			}
		}

		if upToLatest {
			reversedVersions = append(reversedVersions, versionFromRelease(release))
		}
	}

	if !upToLatest {
		// current version was removed; start over from latest
		reversedVersions = append(
			reversedVersions,
			versionFromRelease(filteredReleases[len(filteredReleases)-1]),
		)
	}

	return reversedVersions, nil
}
