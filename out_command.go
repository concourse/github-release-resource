package resource

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v66/github"
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

	tag = request.Params.TagPrefix + tag

	var body string
	bodySpecified := false
	if request.Params.BodyPath != "" {
		bodySpecified = true

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

	draft := request.Source.Drafts
	prerelease := false
	if request.Source.PreRelease == true && request.Source.Release == false {
		prerelease = request.Source.PreRelease
	}

	generateReleaseNotes := request.Params.GenerateReleaseNotes

	release := &github.RepositoryRelease{
		Name:                 github.String(name),
		TagName:              github.String(tag),
		Body:                 github.String(body),
		Draft:                github.Bool(draft),
		Prerelease:           github.Bool(prerelease),
		TargetCommitish:      github.String(targetCommitish),
		GenerateReleaseNotes: github.Bool(generateReleaseNotes),
	}

	existingReleases, err := c.github.ListReleases()
	if err != nil {
		return OutResponse{}, err
	}

	var existingRelease *github.RepositoryRelease
	for _, e := range existingReleases {
		if e.TagName != nil && *e.TagName == tag {
			existingRelease = e
			break
		}
	}

	if existingRelease != nil {
		releaseAssets, err := c.github.ListReleaseAssets(*existingRelease)
		if err != nil {
			return OutResponse{}, err
		}

		existingRelease.Name = github.String(name)
		existingRelease.TargetCommitish = github.String(targetCommitish)
		existingRelease.Draft = github.Bool(draft)
		existingRelease.Prerelease = github.Bool(prerelease)

		if bodySpecified {
			existingRelease.Body = github.String(body)
		} else {
			existingRelease.Body = nil
		}

		for _, asset := range releaseAssets {
			fmt.Fprintf(c.writer, "clearing existing asset: %s\n", *asset.Name)

			err := c.github.DeleteReleaseAsset(*asset)
			if err != nil {
				return OutResponse{}, err
			}
		}

		fmt.Fprintf(c.writer, "updating release %s\n", name)

		release, err = c.github.UpdateRelease(*existingRelease)
		if err != nil {
			return OutResponse{}, err
		}
	} else {
		fmt.Fprintf(c.writer, "creating release %s\n", name)
		release, err = c.github.CreateRelease(*release)
		if err != nil {
			return OutResponse{}, err
		}
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
			err := c.upload(release, filePath)
			if err != nil {
				return OutResponse{}, err
			}
		}
	}

	return OutResponse{
		Version:  versionFromRelease(release),
		Metadata: metadataFromRelease(release, ""),
	}, nil
}

func (c *OutCommand) fileContents(path string) (string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}

func (c *OutCommand) upload(release *github.RepositoryRelease, filePath string) error {
	fmt.Fprintf(c.writer, "uploading %s\n", filePath)

	name := filepath.Base(filePath)

	var retryErr error
	for i := 0; i < 10; i++ {
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}

		defer file.Close()

		retryErr = c.github.UploadReleaseAsset(*release, name, file)
		if retryErr == nil {
			break
		}

		assets, err := c.github.ListReleaseAssets(*release)
		if err != nil {
			return err
		}

		for _, asset := range assets {
			if asset.Name != nil && *asset.Name == name {
				err = c.github.DeleteReleaseAsset(*asset)
				if err != nil {
					return err
				}
				break
			}
		}
	}

	if retryErr != nil {
		return retryErr
	}

	return nil
}
