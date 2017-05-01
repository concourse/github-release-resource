package resource

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"crypto/tls"

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
	err := os.MkdirAll(destDir, 0755)
	if err != nil {
		return InResponse{}, err
	}

	var foundRelease *github.RepositoryRelease

	if request.Version.Tag != "" {
		foundRelease, err = c.github.GetReleaseByTag(request.Version.Tag)
	} else {
		id, _ := strconv.Atoi(request.Version.ID)
		foundRelease, err = c.github.GetRelease(id)
	}
	if err != nil {
		return InResponse{}, err
	}

	if foundRelease == nil {
		return InResponse{}, errors.New("no releases")
	}

	if foundRelease.TagName != nil && *foundRelease.TagName != "" {
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

		if foundRelease.Body != nil && *foundRelease.Body != "" {
			body := *foundRelease.Body
			bodyPath := filepath.Join(destDir, "body")
			err = ioutil.WriteFile(bodyPath, []byte(body), 0644)
			if err != nil {
				return InResponse{}, err
			}
		}

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

		err := c.downloadAsset(asset, path)
		if err != nil {
			return InResponse{}, err
		}
	}

	if request.Params.IncludeSourceTarball {
		u, err := c.github.GetTarballLink(request.Version.Tag)
		if err != nil {
			return InResponse{}, err
		}
		fmt.Fprintln(c.writer, "downloading source tarball to source.tar.gz")
		if err := c.downloadFile(u.String(), filepath.Join(destDir, "source.tar.gz"), request.Source.SkipSSLValidation); err != nil {
			return InResponse{}, err
		}
	}

	if request.Params.IncludeSourceZip {
		u, err := c.github.GetZipballLink(request.Version.Tag)
		if err != nil {
			return InResponse{}, err
		}
		fmt.Fprintln(c.writer, "downloading source zip to source.zip")
		if err := c.downloadFile(u.String(), filepath.Join(destDir, "source.zip"), request.Source.SkipSSLValidation); err != nil {
			return InResponse{}, err
		}
	}

	return InResponse{
		Version:  versionFromRelease(foundRelease),
		Metadata: metadataFromRelease(foundRelease),
	}, nil
}

func (c *InCommand) downloadAsset(asset *github.ReleaseAsset, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	content, err := c.github.DownloadReleaseAsset(*asset)
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

func (c *InCommand) downloadFile(url, destPath string, skipSsl bool) error {
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipSsl},
	}
	httpClient := &http.Client{Transport: tr}

	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file `%s`: HTTP status %d", filepath.Base(destPath), resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
