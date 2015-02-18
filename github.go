package resource

import (
	"os"

	"code.google.com/p/goauth2/oauth"

	"github.com/google/go-github/github"
)

//go:generate counterfeiter . GitHub

type GitHub interface {
	ListReleases() ([]github.RepositoryRelease, error)
	CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error)

	ListReleaseAssets(release *github.RepositoryRelease) ([]github.ReleaseAsset, error)
	UploadReleaseAsset(release *github.RepositoryRelease, name string, file *os.File) error
}

type GitHubClient struct {
	client *github.Client

	user       string
	repository string
}

func NewGitHubClient(source Source) *GitHubClient {
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

func (g *GitHubClient) ListReleases() ([]github.RepositoryRelease, error) {
	releases, _, err := g.client.Repositories.ListReleases(g.user, g.repository, nil)
	if err != nil {
		return []github.RepositoryRelease{}, err
	}

	return releases, nil
}

func (g *GitHubClient) CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	createdRelease, _, err := g.client.Repositories.CreateRelease(g.user, g.repository, release)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	return createdRelease, nil
}

func (g *GitHubClient) ListReleaseAssets(release *github.RepositoryRelease) ([]github.ReleaseAsset, error) {
	assets, _, err := g.client.Repositories.ListReleaseAssets(g.user, g.repository, *release.ID, nil)
	if err != nil {
		return []github.ReleaseAsset{}, nil
	}

	return assets, nil
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
