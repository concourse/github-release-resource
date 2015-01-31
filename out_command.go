package resource

import (
	"os"
	"path/filepath"

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

	release := &github.RepositoryRelease{}
	createdRelease, err := c.github.CreateRelease(release)
	if err != nil {
		return OutResponse{}, err
	}

	for _, filePath := range params.Globs {
		file, err := os.Open(filePath)
		if err != nil {
			return OutResponse{}, err
		}

		name := filepath.Base(filePath)
		err = c.github.UploadReleaseAsset(createdRelease, name, file)
		if err != nil {
			return OutResponse{}, err
		}

		file.Close()
	}

	return OutResponse{}, nil
}
