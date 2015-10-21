package resource

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/zachgersh/go-github/github"
)

type OutCommand struct {
	github GitHub
	writer io.Writer
}

func NewOutCommand(github GitHub, writer io.Writer) *OutCommand {
	return &OutCommand{
		github: github,
		writer: writer,
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

	targetCommitish := ""
	if request.Params.CommitishPath != "" {
		targetCommitish, err = c.fileContents(filepath.Join(sourceDir, request.Params.CommitishPath))
		if err != nil {
			return OutResponse{}, err
		}
	}

	draft := request.Params.Draft

	release := &github.RepositoryRelease{
		Name:            github.String(name),
		TagName:         github.String(tag),
		Body:            github.String(body),
		Draft:           github.Bool(draft),
		TargetCommitish: github.String(targetCommitish),
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
		existingRelease.TargetCommitish = github.String(targetCommitish)

		for _, asset := range existingRelease.Assets {
			fmt.Fprintf(c.writer, "clearing existing asset: %s\n", *asset.Name)

			err := c.github.DeleteReleaseAsset(asset)
			if err != nil {
				return OutResponse{}, err
			}
		}

		fmt.Fprintf(c.writer, "updating release %s\n", name)

		release, err = c.github.UpdateRelease(*existingRelease)
	} else {
		fmt.Fprintf(c.writer, "creating release %s\n", name)
		release, err = c.github.CreateRelease(*release)
	}

	if err != nil {
		return OutResponse{}, err
	}

	for _, fileGlob := range params.Globs {
		matches, err := filepath.Glob(filepath.Join(sourceDir, fileGlob))
		if err != nil {
			return OutResponse{}, err
		}

		if len(matches) == 0 {
			return OutResponse{}, fmt.Errorf("could not find file that matches glob '%s'", fileGlob)
		}

		for _, filePath := range matches {
			file, err := os.Open(filePath)
			if err != nil {
				return OutResponse{}, err
			}

			fmt.Fprintf(c.writer, "uploading %s\n", filePath)

			name := filepath.Base(filePath)
			err = c.github.UploadReleaseAsset(*release, name, file)
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
