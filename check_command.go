package resource

import (
	"errors"

	"github.com/google/go-github/github"
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
		return []Version{}, errors.New("repository had no releases")
	}

	latestVersion := *releases[0].TagName

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
	for _, release := range reverse(releases) {
		version := *release.TagName

		if upToLatest {
			reversedVersions = append(reversedVersions, Version{Tag: version})
		} else {
			upToLatest = request.Version.Tag == version
		}
	}

	return reversedVersions, nil
}

func reverse(s []github.RepositoryRelease) []github.RepositoryRelease {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}

	return s
}
