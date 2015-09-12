package resource

import (
	"sort"

	"github.com/blang/semver"
	"github.com/zachgersh/go-github/github"
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

	sort.Sort(byVersion(releases))

	var filteredReleases []github.RepositoryRelease

	for _, release := range releases {
		draft := *release.Draft
		if !draft {
			filteredReleases = append(filteredReleases, release)
		}
	}

	latestVersion := *filteredReleases[len(filteredReleases)-1].TagName

	if request.Version.Tag == "" {
		return []Version{
			{Tag: latestVersion},
		}, nil
	}

	if latestVersion == request.Version.Tag {
		return []Version{}, nil
	}

	upToLatest := false
	reversedVersions := []Version{}
	for _, release := range filteredReleases {
		version := *release.TagName

		if upToLatest {
			reversedVersions = append(reversedVersions, Version{Tag: version})
		} else {
			upToLatest = request.Version.Tag == version
		}
	}

	return reversedVersions, nil
}

type byVersion []github.RepositoryRelease

func (r byVersion) Len() int {
	return len(r)
}

func (r byVersion) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r byVersion) Less(i, j int) bool {
	if r[i].TagName == nil || r[j].TagName == nil {
		return false
	}

	first, err := semver.New(determineVersionFromTag(*r[i].TagName))
	if err != nil {
		return true
	}

	second, err := semver.New(determineVersionFromTag(*r[j].TagName))
	if err != nil {
		return false
	}

	return first.LT(*second)
}
