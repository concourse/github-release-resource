package resource

import (
	"os"

	"code.google.com/p/goauth2/oauth"

	"github.com/google/go-github/github"
)

//go:generate counterfeiter . GitHub

type GitHub interface {
	CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error)
	UploadReleaseAsset(release *github.RepositoryRelease, name string, file *os.File) error
}

type GitHubClient struct {
	client *github.Client

	user       string
	repository string
}

func NewGitHubClient(source OutSource) *GitHubClient {
	transport := &oauth.Transport{
		Token: &oauth.Token{
			AccessToken: source.AccessToken,
		},
	}

	client := github.NewClient(transport.Client())

	return &GitHubClient{
		client:     client,
		user:       source.User,
		repository: source.Repository,
	}
}

func (g *GitHubClient) CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	createdRelease, _, err := g.client.Repositories.CreateRelease(g.user, g.repository, release)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	return createdRelease, nil
}

func (g *GitHubClient) UploadReleaseAsset(release *github.RepositoryRelease, name string, file *os.File) error {
	_, _, err := g.client.Repositories.UploadReleaseAsset(
		g.user,
		g.repository,
		*release.ID,
		&github.UploadOptions{
			Name: name,
		},
		file,
	)

	return err
}
