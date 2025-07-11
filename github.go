package resource

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_git_hub.go . GitHub
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
	ResolveTagToCommitSHA(tag string) (string, error)
}

type GitHubClient struct {
	client       *github.Client
	clientV4     *githubv4.Client
	isEnterprise bool

	owner       string
	repository  string
	accessToken string
}

func NewGitHubClient(source Source) (*GitHubClient, error) {
	httpClient := &http.Client{}
	ctx := context.TODO()

	if source.Insecure {
		httpClient.Transport = &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	if source.AccessToken != "" {
		var err error
		httpClient, err = oauthClient(ctx, source)
		if err != nil {
			return nil, err
		}
	}

	client := github.NewClient(httpClient)

	clientV4 := githubv4.NewClient(httpClient)
	var isEnterprise bool

	if source.GitHubAPIURL != "" {
		var err error
		if !strings.HasSuffix(source.GitHubAPIURL, "/") {
			source.GitHubAPIURL += "/"
		}
		client.BaseURL, err = url.Parse(source.GitHubAPIURL)
		if err != nil {
			return nil, err
		}

		client.UploadURL, err = url.Parse(source.GitHubAPIURL)
		if err != nil {
			return nil, err
		}

		var v4URL string
		if strings.HasSuffix(source.GitHubAPIURL, "/v3/") {
			v4URL = strings.TrimSuffix(source.GitHubAPIURL, "/v3/") + "/graphql"
		} else {
			v4URL = source.GitHubAPIURL + "graphql"
		}
		clientV4 = githubv4.NewEnterpriseClient(v4URL, httpClient)
		isEnterprise = true
	}

	if source.GitHubV4APIURL != "" {
		clientV4 = githubv4.NewEnterpriseClient(source.GitHubV4APIURL, httpClient)
		isEnterprise = true
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
		client:       client,
		clientV4:     clientV4,
		isEnterprise: isEnterprise,
		owner:        owner,
		repository:   source.Repository,
		accessToken:  source.AccessToken,
	}, nil
}

func (g *GitHubClient) ListReleases() ([]*github.RepositoryRelease, error) {
	if g.accessToken != "" {
		if g.isEnterprise {
			return g.listReleasesV4EnterPrice()
		}
		return g.listReleasesV4()
	}
	opt := &github.ListOptions{PerPage: 100}
	var allReleases []*github.RepositoryRelease
	for {
		releases, res, err := g.client.Repositories.ListReleases(context.TODO(), g.owner, g.repository, opt)
		if err != nil {
			return []*github.RepositoryRelease{}, err
		}
		allReleases = append(allReleases, releases...)
		if res.NextPage == 0 {
			err = res.Body.Close()
			if err != nil {
				return nil, err
			}
			break
		}
		opt.Page = res.NextPage
	}

	return allReleases, nil
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
	release, res, err := g.client.Repositories.GetRelease(context.TODO(), g.owner, g.repository, int64(id))
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
	opt := &github.ListOptions{PerPage: 100}
	var allAssets []*github.ReleaseAsset
	for {
		assets, res, err := g.client.Repositories.ListReleaseAssets(context.TODO(), g.owner, g.repository, *release.ID, opt)
		if err != nil {
			return []*github.ReleaseAsset{}, err
		}
		allAssets = append(allAssets, assets...)
		if res.NextPage == 0 {
			err = res.Body.Close()
			if err != nil {
				return nil, err
			}
			break
		}
		opt.Page = res.NextPage
	}

	return allAssets, nil
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
	bodyReader, redirectURL, err := g.client.Repositories.DownloadReleaseAsset(context.TODO(), g.owner, g.repository, *asset.ID, nil)
	if err != nil {
		return nil, err
	}

	if redirectURL == "" {
		return bodyReader, err
	}

	req, err := g.client.NewRequest("GET", redirectURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	if g.accessToken != "" && req.URL.Host == g.client.BaseURL.Host {
		req.Header.Set("Authorization", "Bearer "+g.accessToken)
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		resp.Body.Close()
		return nil, fmt.Errorf("redirect URL %q responded with bad status code: %d", redirectURL, resp.StatusCode)
	}

	return resp.Body, nil
}

func (g *GitHubClient) GetTarballLink(tag string) (*url.URL, error) {
	opt := &github.RepositoryContentGetOptions{Ref: tag}
	u, res, err := g.client.Repositories.GetArchiveLink(context.TODO(), g.owner, g.repository, github.Tarball, opt, 10)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func (g *GitHubClient) GetZipballLink(tag string) (*url.URL, error) {
	opt := &github.RepositoryContentGetOptions{Ref: tag}
	u, res, err := g.client.Repositories.GetArchiveLink(context.TODO(), g.owner, g.repository, github.Zipball, opt, 10)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	return u, nil
}

func (g *GitHubClient) ResolveTagToCommitSHA(tagName string) (string, error) {
	ref, res, err := g.client.Git.GetRef(context.TODO(), g.owner, g.repository, "tags/"+tagName)
	if err != nil {
		return "", err
	}

	res.Body.Close()

	// Lightweight tag
	if *ref.Object.Type == "commit" {
		return *ref.Object.SHA, nil
	}

	// Fail if we're not pointing to a annotated tag
	if *ref.Object.Type != "tag" {
		return "", fmt.Errorf("could not resolve tag %q to commit: returned type is not 'commit' or 'tag'", tagName)
	}

	// Resolve tag to commit sha
	tag, res, err := g.client.Git.GetTag(context.TODO(), g.owner, g.repository, *ref.Object.SHA)
	if err != nil {
		return "", err
	}

	res.Body.Close()

	if *tag.Object.Type != "commit" {
		return "", fmt.Errorf("could not resolve tag %q to commit: returned type is not 'commit'", tagName)
	}

	return *tag.Object.SHA, nil
}

func oauthClient(ctx context.Context, source Source) (*http.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: source.AccessToken,
	})

	oauthClient := oauth2.NewClient(ctx, ts)

	githubHTTPClient := &http.Client{
		Transport: oauthClient.Transport,
	}

	return githubHTTPClient, nil
}
