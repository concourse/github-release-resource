package resource

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

//go:generate counterfeiter . GitHub

type GitHub interface {
	ListReleases() ([]*github.RepositoryRelease, error)
	GetReleaseByTag(tag string) (*github.RepositoryRelease, error)
	GetRelease(id int) (*github.RepositoryRelease, error)
	CreateRelease(release github.RepositoryRelease) (*github.RepositoryRelease, error)
	UpdateRelease(release github.RepositoryRelease) (*github.RepositoryRelease, error)

	ListReleaseAssets(release github.RepositoryRelease) ([]*github.ReleaseAsset, error)
	UploadReleaseAsset(release github.RepositoryRelease, name string, file *os.File) error
	DeleteReleaseAsset(asset github.ReleaseAsset) error
	DownloadReleaseAsset(asset github.ReleaseAsset) (io.ReadCloser, error)

	GetTarballLink(tag string) (*url.URL, error)
	GetZipballLink(tag string) (*url.URL, error)
}

type GitHubClient struct {
	client *github.Client

	user       string
	repository string
}

func NewGitHubClient(source Source) (*GitHubClient, error) {
	var client *github.Client

	if source.AccessToken == "" {
		client = github.NewClient(nil)
	} else {
		var err error
		client, err = oauthClient(source)
		if err != nil {
			return nil, err
		}
	}

	if source.GitHubAPIURL != "" {
		var err error
		client.BaseURL, err = url.Parse(source.GitHubAPIURL)
		if err != nil {
			return nil, err
		}

		client.UploadURL, err = url.Parse(source.GitHubAPIURL)
		if err != nil {
			return nil, err
		}
	}

	if source.GitHubUploadsURL != "" {
		var err error
		client.UploadURL, err = url.Parse(source.GitHubUploadsURL)
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

func (g *GitHubClient) ListReleases() ([]*github.RepositoryRelease, error) {
	releases, res, err := g.client.Repositories.ListReleases(g.user, g.repository, nil)
	if err != nil {
		return []*github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func (g *GitHubClient) GetReleaseByTag(tag string) (*github.RepositoryRelease, error) {
	release, res, err := g.client.Repositories.GetReleaseByTag(g.user, g.repository, tag)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *GitHubClient) GetRelease(id int) (*github.RepositoryRelease, error) {
	release, res, err := g.client.Repositories.GetRelease(g.user, g.repository, id)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return release, nil
}

func (g *GitHubClient) CreateRelease(release github.RepositoryRelease) (*github.RepositoryRelease, error) {
	createdRelease, res, err := g.client.Repositories.CreateRelease(g.user, g.repository, &release)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return createdRelease, nil
}

func (g *GitHubClient) UpdateRelease(release github.RepositoryRelease) (*github.RepositoryRelease, error) {
	if release.ID == nil {
		return nil, errors.New("release did not have an ID: has it been saved yet?")
	}

	updatedRelease, res, err := g.client.Repositories.EditRelease(g.user, g.repository, *release.ID, &release)
	if err != nil {
		return &github.RepositoryRelease{}, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return updatedRelease, nil
}

func (g *GitHubClient) ListReleaseAssets(release github.RepositoryRelease) ([]*github.ReleaseAsset, error) {
	assets, res, err := g.client.Repositories.ListReleaseAssets(g.user, g.repository, *release.ID, nil)
	if err != nil {
		return nil, err
	}

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return assets, nil
}

func (g *GitHubClient) UploadReleaseAsset(release github.RepositoryRelease, name string, file *os.File) error {
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

func (g *GitHubClient) DownloadReleaseAsset(asset github.ReleaseAsset) (io.ReadCloser, error) {
	res, redir, err := g.client.Repositories.DownloadReleaseAsset(g.user, g.repository, *asset.ID)
	if err != nil {
		return nil, err
	}

	if redir != "" {
		resp, err := http.Get(redir)
		if err != nil {
			return nil, err
		}

		return resp.Body, nil
	}

	return res, err
}

func (g *GitHubClient) GetTarballLink(tag string) (*url.URL, error) {
	opt := &github.RepositoryContentGetOptions{Ref: tag}
	u, res, err := g.client.Repositories.GetArchiveLink(g.user, g.repository, github.Tarball, opt)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func (g *GitHubClient) GetZipballLink(tag string) (*url.URL, error) {
	opt := &github.RepositoryContentGetOptions{Ref: tag}
	u, res, err := g.client.Repositories.GetArchiveLink(g.user, g.repository, github.Zipball, opt)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func oauthClient(source Source) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: source.AccessToken,
	})

	oauthClient := oauth2.NewClient(oauth2.NoContext, ts)

	githubHTTPClient := &http.Client{
		Transport: oauthClient.Transport,
	}

	return github.NewClient(githubHTTPClient), nil
}
