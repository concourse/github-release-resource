package resource_test

import (
	"testing"

	"github.com/google/go-github/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGithubReleaseResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github Release Resource Suite")
}

func newRepositoryRelease(id int, version string) *github.RepositoryRelease {
	return &github.RepositoryRelease{
		TagName:    github.String(version),
		Draft:      github.Bool(false),
		Prerelease: github.Bool(false),
		ID:         github.Int(id),
	}
}

func newPreReleaseRepositoryRelease(id int, version string) *github.RepositoryRelease {
	return &github.RepositoryRelease{
		TagName:    github.String(version),
		Draft:      github.Bool(false),
		Prerelease: github.Bool(true),
		ID:         github.Int(id),
	}
}
func newDraftRepositoryRelease(id int, version string) *github.RepositoryRelease {
	return &github.RepositoryRelease{
		TagName:    github.String(version),
		Draft:      github.Bool(true),
		Prerelease: github.Bool(false),
		ID:         github.Int(id),
	}
}

func newDraftWithNilTagRepositoryRelease(id int) *github.RepositoryRelease {
	return &github.RepositoryRelease{
		Draft:      github.Bool(true),
		Prerelease: github.Bool(false),
		ID:         github.Int(id),
	}
}
