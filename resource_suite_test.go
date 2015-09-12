package resource_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/zachgersh/go-github/github"
)

func TestGithubReleaseResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github Release Resource Suite")
}

func newRepositoryRelease(version string) github.RepositoryRelease {
	draft := false
	return github.RepositoryRelease{
		TagName: github.String(version),
		Draft:   &draft,
	}
}

func newDraftRepositoryRelease(version string) github.RepositoryRelease {
	draft := true
	return github.RepositoryRelease{
		TagName: github.String(version),
		Draft:   &draft,
	}
}
