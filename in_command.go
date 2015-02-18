package resource

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/google/go-github/github"
)

type InCommand struct {
	github GitHub
	writer io.Writer
}

func NewInCommand(github GitHub, writer io.Writer) *InCommand {
	return &InCommand{
		github: github,
		writer: writer,
	}
}

func (c *InCommand) Run(destDir string, request InRequest) (InResponse, error) {
	releases, err := c.github.ListReleases()
	if err != nil {
		return InResponse{}, nil
	}

	sort.Sort(byVersion(releases))

	if len(releases) == 0 {
		return InResponse{}, errors.New("no releases")
	}

	var foundRelease *github.RepositoryRelease

	if request.Version == nil {
		foundRelease = &releases[len(releases)-1]
	} else {
		for _, release := range releases {
			if *release.TagName == request.Version.Tag {
				foundRelease = &release
				break
			}
		}
	}

	if foundRelease == nil {
		return InResponse{}, fmt.Errorf("could not find release with tag: %s", request.Version.Tag)
	}

	assets, err := c.github.ListReleaseAssets(foundRelease)
	if err != nil {
		return InResponse{}, nil
	}

	for _, asset := range assets {
		url := *asset.BrowserDownloadURL
		path := filepath.Join(destDir, *asset.Name)

		var matchFound bool
		if len(request.Params.Globs) == 0 {
			matchFound = true
		} else {
			for _, glob := range request.Params.Globs {
				matches, err := filepath.Match(glob, *asset.Name)
				if err != nil {
					return InResponse{}, err
				}

				if matches {
					matchFound = true
					break
				}
			}
		}

		if !matchFound {
			continue
		}

		fmt.Fprintf(c.writer, "downloading asset: %s\n", *asset.Name)
		err := c.downloadFile(url, path)
		if err != nil {
			return InResponse{}, nil
		}
	}

	return InResponse{
		Version: Version{
			Tag: *foundRelease.TagName,
		},
		Metadata: []MetadataPair{
			{Name: "name", Value: *foundRelease.Name, URL: *foundRelease.HTMLURL},
			{Name: "url", Value: *foundRelease.HTMLURL},
			{Name: "body", Value: *foundRelease.Body, Markdown: true},
		},
	}, nil
}

func (c *InCommand) downloadFile(url, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
