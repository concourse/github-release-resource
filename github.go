package resource

import (
	"errors"
	"net/url"
	"os"

	"code.google.com/p/goauth2/oauth"

	"github.com/google/go-github/github"
)

//go:generate counterfeiter . GitHub

type GitHub interface {
	ListReleases() ([]github.RepositoryRelease, error)
	CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error)
	UpdateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error)

	ListReleaseAssets(release *github.RepositoryRelease) ([]github.ReleaseAsset, error)
	UploadReleaseAsset(release *github.RepositoryRelease, name string, file *os.File) error
	DeleteReleaseAsset(asset github.ReleaseAsset) error
}

type GitHubClient struct {
	client *github.Client

	user       string
	repository string
}

func NewGitHubClient(source Source) (*GitHubClient, error) {
	transport := &oauth.Transport{
		Token: &oauth.Token{
			AccessToken: source.AccessToken,
		},
	}

	var client *github.Client

	if transport.Token.AccessToken == "" {
		client = github.NewClient(nil)
	} else {
		client = github.NewClient(transport.Client())
	}

	if source.GitHubAPIURL != "" {
		var err error
		client.BaseURL, err = url.Parse(source.GitHubAPIURL)
		if err != nil {
			return nil, err
		}
	}

	return &GitHubClient{
		client:     client,
		user:       source.User,
		repository: source.Repository,
	}, nil
}

func (g *GitHubClient) ListReleases() ([]github.RepositoryRelease, error) {
	releases, res, err := g.client.Repositories.ListReleases(g.user, g.repository, nil)
	if err != nil {
		return []github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (g *GitHubClient) CreateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	createdRelease, res, err := g.client.Repositories.CreateRelease(g.user, g.repository, release)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return createdRelease, nil
}

func (g *GitHubClient) UpdateRelease(release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	if release.ID == nil {
		return nil, errors.New("release did not have an ID: has it been saved yet?")
	}

	updatedRelease, res, err := g.client.Repositories.EditRelease(g.user, g.repository, *release.ID, release)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return updatedRelease, nil
}

func (g *GitHubClient) ListReleaseAssets(release *github.RepositoryRelease) ([]github.ReleaseAsset, error) {
	assets, res, err := g.client.Repositories.ListReleaseAssets(g.user, g.repository, *release.ID, nil)
	if err != nil {
		return []github.ReleaseAsset{}, nil
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return assets, nil
}

func (g *GitHubClient) UploadReleaseAsset(release *github.RepositoryRelease, name string, file *os.File) error {
	_, res, err := g.client.Repositories.UploadReleaseAsset(
		g.user,
		g.repository,
		*release.ID,
		&github.UploadOptions{
			Name: name,
		},
		file,
	)
	if err != nil {
		return err
	}

	return res.Body.Close()
}

func (g *GitHubClient) DeleteReleaseAsset(asset github.ReleaseAsset) error {
	res, err := g.client.Repositories.DeleteReleaseAsset(g.user, g.repository, *asset.ID)
	if err != nil {
		return err
	}

	return res.Body.Close()
}
