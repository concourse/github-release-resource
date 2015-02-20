package resource

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
)

type OutCommand struct {
	github GitHub
}

func NewOutCommand(github GitHub) *OutCommand {
	return &OutCommand{
		github: github,
	}
}

func (c *OutCommand) Run(sourceDir string, request OutRequest) (OutResponse, error) {
	params := request.Params

	name, err := c.fileContents(filepath.Join(sourceDir, request.Params.NamePath))
	if err != nil {
		return OutResponse{}, err
	}

	tag, err := c.fileContents(filepath.Join(sourceDir, request.Params.TagPath))
	if err != nil {
		return OutResponse{}, err
	}

	body := ""
	if request.Params.BodyPath != "" {
		body, err = c.fileContents(filepath.Join(sourceDir, request.Params.BodyPath))
		if err != nil {
			return OutResponse{}, err
		}
	}

	release := &github.RepositoryRelease{
		Name:    github.String(name),
		TagName: github.String(tag),
		Body:    github.String(body),
	}

	existingReleases, err := c.github.ListReleases()
	if err != nil {
		return OutResponse{}, err
	}

	var existingRelease *github.RepositoryRelease
	for _, e := range existingReleases {
		if *e.TagName == tag {
			existingRelease = &e
			break
		}
	}

	if existingRelease != nil {
		existingRelease.Name = github.String(name)
		existingRelease.Body = github.String(body)

		release, err = c.github.UpdateRelease(existingRelease)
	} else {
		release, err = c.github.CreateRelease(release)
	}

	if err != nil {
		return OutResponse{}, err
	}

	for _, fileGlob := range params.Globs {
		matches, err := filepath.Glob(filepath.Join(sourceDir, fileGlob))
		if err != nil {
			return OutResponse{}, err
		}

		for _, filePath := range matches {
			file, err := os.Open(filePath)
			if err != nil {
				return OutResponse{}, err
			}

			name := filepath.Base(filePath)
			err = c.github.UploadReleaseAsset(release, name, file)
			if err != nil {
				return OutResponse{}, err
			}

			file.Close()
		}
	}

	return OutResponse{
		Version: Version{
			Tag: tag,
		},
		Metadata: metadataFromRelease(release),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}
