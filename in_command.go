package resource

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/zachgersh/go-github/github"
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
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return InResponse{}, err
	}

	var foundRelease *github.RepositoryRelease

	if request.Version == nil {
		var err error

		foundRelease, err = c.github.LatestRelease()
		if err != nil {
			return InResponse{}, err
		}
	} else {
		var err error

		foundRelease, err = c.github.GetReleaseByTag(request.Version.Tag)
		if err != nil {
			return InResponse{}, err
		}
	}

	if foundRelease == nil {
		return InResponse{}, errors.New("no releases")
	}

	tagPath := filepath.Join(destDir, "tag")
	err = ioutil.WriteFile(tagPath, []byte(*foundRelease.TagName), 0644)
	if err != nil {
		return InResponse{}, err
	}

	version := determineVersionFromTag(*foundRelease.TagName)
	versionPath := filepath.Join(destDir, "version")
	err = ioutil.WriteFile(versionPath, []byte(version), 0644)
	if err != nil {
		return InResponse{}, err
	}

	assets, err := c.github.ListReleaseAssets(*foundRelease)
	if err != nil {
		return InResponse{}, err
	}

	for _, asset := range assets {
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

		err := c.downloadFile(asset, path)
		if err != nil {
			return InResponse{}, err
		}
	}

	return InResponse{
		Version: Version{
			Tag: *foundRelease.TagName,
		},
		Metadata: metadataFromRelease(foundRelease),
	}, nil
}

func (c *InCommand) downloadFile(asset github.ReleaseAsset, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	content, err := c.github.DownloadReleaseAsset(asset)
	if err != nil {
		return err
	}
	defer content.Close()

	_, err = io.Copy(out, content)
	if err != nil {
		return err
	}

	return nil
}
