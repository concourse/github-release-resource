package resource

import (
	"sort"

	"github.com/Masterminds/semver"
	"github.com/cppforlife/go-semi-semantic/version"
	"github.com/google/go-github/v66/github"
)

type CheckCommand struct {
	github GitHub
}

func NewCheckCommand(github GitHub) *CheckCommand {
	return &CheckCommand{
		github: github,
	}
}

func SortByVersion(releases []*github.RepositoryRelease, versionParser *versionParser) {
	sort.Slice(releases, func(i, j int) bool {
		first, err := version.NewVersionFromString(versionParser.parse(*releases[i].TagName))
		if err != nil {
			return true
		}

		second, err := version.NewVersionFromString(versionParser.parse(*releases[j].TagName))
		if err != nil {
			return false
		}

		return first.IsLt(second)
	})
}

func SortByTimestamp(releases []*github.RepositoryRelease) {
	sort.Slice(releases, func(i, j int) bool {
		a := releases[i]
		b := releases[j]
		return getTimestamp(a).Before(getTimestamp(b))
	})
}

func (c *CheckCommand) Run(request CheckRequest) ([]Version, error) {
	releases, err := c.github.ListReleases()
	if err != nil {
		return []Version{}, err
	}

	if len(releases) == 0 {
		return []Version{}, nil
	}

	orderByTime := false
	if request.Source.OrderBy == "time" {
		orderByTime = true
	}

	var filteredReleases []*github.RepositoryRelease

	versionParser, err := newVersionParser(request.Source.TagFilter)
	if err != nil {
		return []Version{}, err
	}

	var constraint *semver.Constraints
	if request.Source.SemverConstraint != "" {
		constraint, err = semver.NewConstraint(request.Source.SemverConstraint)
		if err != nil {
			return []Version{}, err
		}
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

		if constraint != nil {
			if release.TagName == nil {
				// Release has no tag, so certainly isn't a valid semver
				continue
			}
			version, err := semver.NewVersion(versionParser.parse(*release.TagName))
			if err != nil {
				// Release is not tagged with a valid semver
				continue
			}
			if !constraint.Check(version) {
				// Valid semver, but does not satisfy constraint
				continue
			}
		}

		if orderByTime {
			// We won't do anything with the tags, so just make sure the filter matches the tag.
			var tag string
			if release.TagName != nil {
				tag = *release.TagName
			}
			if !versionParser.re.MatchString(tag) {
				continue
			}
			// We don't expect any releases with a missing (zero) timestamp,
			// but we skip those just in case, since the data type includes them
			if getTimestamp(release).IsZero() {
				continue
			}
		} else {
			// We will sort by versions parsed out of tags, so make sure we parse successfully.
			if release.TagName == nil {
				continue
			}
			if _, err := version.NewVersionFromString(versionParser.parse(*release.TagName)); err != nil {
				continue
			}
		}

		filteredReleases = append(filteredReleases, release)
	}

	// If there are no valid releases, output an empty list.

	if len(filteredReleases) == 0 {
		return []Version{}, nil
	}

	// Sort releases by time or by version

	if orderByTime {
		SortByTimestamp(filteredReleases)
	} else {
		SortByVersion(filteredReleases, &versionParser)
	}

	// If request has no version, output the latest release

	latestRelease := filteredReleases[len(filteredReleases)-1]

	if (request.Version == Version{}) {
		return []Version{
			versionFromRelease(latestRelease),
		}, nil
	}

	// Find first release equal or later than the current version

	var firstIncludedReleaseIndex int = -1

	if orderByTime {
		// Only search if request has a timestamp
		if !request.Version.Timestamp.IsZero() {
			firstIncludedReleaseIndex = sort.Search(len(filteredReleases), func(i int) bool {
				release := filteredReleases[i]
				return !getTimestamp(release).Before(request.Version.Timestamp)
			})
		}
	} else {
		requestVersion, err := version.NewVersionFromString(versionParser.parse(request.Version.Tag))
		if err == nil {
			firstIncludedReleaseIndex = sort.Search(len(filteredReleases), func(i int) bool {
				release := filteredReleases[i]
				releaseVersion, err := version.NewVersionFromString(versionParser.parse(*release.TagName))
				if err != nil {
					return false
				}
				return !releaseVersion.IsLt(requestVersion)
			})
		}
	}

	// Output all releases equal or later than the current version,
	// or just the latest release if there are no such releases.

	outputVersions := []Version{}

	if firstIncludedReleaseIndex >= 0 && firstIncludedReleaseIndex < len(filteredReleases) {
		// Found first release >= current version, so output this and all the following release versions
		for i := firstIncludedReleaseIndex; i < len(filteredReleases); i++ {
			outputVersions = append(outputVersions, versionFromRelease(filteredReleases[i]))
		}
	} else {
		// No release >= current version, so output the latest release version
		outputVersions = append(
			outputVersions,
			versionFromRelease(filteredReleases[len(filteredReleases)-1]),
		)
	}

	return outputVersions, nil
}
