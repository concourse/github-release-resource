package resource

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"golang.org/x/oauth2"

	"context"

	"github.com/google/go-github/github"
)

//go:generate counterfeiter -o fakes/fake_git_hub.go . GitHub

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

	owner      string
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

	owner := source.Owner
	if source.User != "" {
		owner = source.User
	}

	return &GitHubClient{
		client:     client,
		owner:      owner,
		repository: source.Repository,
	}, nil
}

func (g *GitHubClient) ListReleases() ([]*github.RepositoryRelease, error) {
	releases, res, err := g.client.Repositories.ListReleases(context.TODO(), g.owner, g.repository, nil)
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
	release, res, err := g.client.Repositories.GetReleaseByTag(context.TODO(), g.owner, g.repository, tag)
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
	release, res, err := g.client.Repositories.GetRelease(context.TODO(), g.owner, g.repository, id)
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
	createdRelease, res, err := g.client.Repositories.CreateRelease(context.TODO(), g.owner, g.repository, &release)
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

	updatedRelease, res, err := g.client.Repositories.EditRelease(context.TODO(), g.owner, g.repository, *release.ID, &release)
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
	assets, res, err := g.client.Repositories.ListReleaseAssets(context.TODO(), g.owner, g.repository, *release.ID, nil)
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
		context.TODO(),
		g.owner,
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
	res, err := g.client.Repositories.DeleteReleaseAsset(context.TODO(), g.owner, g.repository, *asset.ID)
	if err != nil {
		return err
	}

	return res.Body.Close()
}

func (g *GitHubClient) DownloadReleaseAsset(asset github.ReleaseAsset) (io.ReadCloser, error) {
	res, redir, err := g.client.Repositories.DownloadReleaseAsset(context.TODO(), g.owner, g.repository, *asset.ID)
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
	u, res, err := g.client.Repositories.GetArchiveLink(context.TODO(), g.owner, g.repository, github.Tarball, opt)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func (g *GitHubClient) GetZipballLink(tag string) (*url.URL, error) {
	opt := &github.RepositoryContentGetOptions{Ref: tag}
	u, res, err := g.client.Repositories.GetArchiveLink(context.TODO(), g.owner, g.repository, github.Zipball, opt)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func filterReleasesByTagName(releases []*github.RepositoryRelease, tagNameRegex string) ([]*github.RepositoryRelease, error) {
	var matchedReleases []*github.RepositoryRelease
	for _, release := range releases {
		if release.TagName == nil || *release.TagName == "" {
			continue
		}
		if matched, err := regexp.MatchString(tagNameRegex, *release.TagName); err == nil {
			if matched {
				matchedReleases = append(matchedReleases, release)
			}
		} else {
			return nil, err
		}
	}
	return matchedReleases, nil
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
