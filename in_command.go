package resource

import (
	"errors"
	"fmt"
	"github.com/google/go-github/v39/github"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	var commitSHA string

	id, _ := strconv.Atoi(request.Version.ID)
	foundRelease, err = c.github.GetRelease(id)
	if err != nil {
		foundRelease, err = c.github.GetReleaseByTag(request.Version.Tag)
		if err != nil {
			return InResponse{}, err
		}
	}

	if foundRelease == nil {
		return InResponse{}, errors.New("no releases")
	}

	if foundRelease.HTMLURL != nil && *foundRelease.HTMLURL != "" {
		urlPath := filepath.Join(destDir, "url")
		err = ioutil.WriteFile(urlPath, []byte(*foundRelease.HTMLURL), 0644)
		if err != nil {
			return InResponse{}, err
		}
	}

	if foundRelease.TagName != nil && *foundRelease.TagName != "" {
		tagPath := filepath.Join(destDir, "tag")
		err = ioutil.WriteFile(tagPath, []byte(*foundRelease.TagName), 0644)
		if err != nil {
			return InResponse{}, err
		}

		versionParser, err := newVersionParser(request.Source.TagFilter)
		if err != nil {
			return InResponse{}, err
		}
		version := versionParser.parse(*foundRelease.TagName)
		versionPath := filepath.Join(destDir, "version")
		err = ioutil.WriteFile(versionPath, []byte(version), 0644)
		if err != nil {
			return InResponse{}, err
		}

		if foundRelease.Draft != nil && !*foundRelease.Draft {
			commitPath := filepath.Join(destDir, "commit_sha")
			commitSHA, err = c.github.ResolveTagToCommitSHA(*foundRelease.TagName)
			if err != nil {
				return InResponse{}, err
			}

			if commitSHA != "" {
				err = ioutil.WriteFile(commitPath, []byte(commitSHA), 0644)
				if err != nil {
					return InResponse{}, err
				}
			}
		}

		if foundRelease.Body != nil && *foundRelease.Body != "" {
			body := *foundRelease.Body
			bodyPath := filepath.Join(destDir, "body")
			err = ioutil.WriteFile(bodyPath, []byte(body), 0644)
			if err != nil {
				return InResponse{}, err
			}

			// Escape UTF-8 characters for Concourse metadata
			body = strconv.QuoteToASCII(*foundRelease.Body)
			body = strings.Replace(body, `\n`, "\n", -1)
			body = strings.Replace(body, `\r`, "\r", -1)
			body = strings.Replace(body, `\t`, "\t", -1)
			foundRelease.Body = &body
		}

		if foundRelease.PublishedAt != nil || foundRelease.CreatedAt != nil {
			timestampPath := filepath.Join(destDir, "timestamp")
			timestamp, err := getTimestamp(foundRelease).MarshalText()
			if err != nil {
				return InResponse{}, err
			}
			err = ioutil.WriteFile(timestampPath, timestamp, 0644)
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
		state := asset.State
		if state == nil || *state != "uploaded" {
			continue
		}

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

	if request.Params.IncludeSourceTarball && foundRelease.TagName != nil {
		u, err := c.github.GetTarballLink(*foundRelease.TagName)
		if err != nil {
			return InResponse{}, err
		}
		fmt.Fprintln(c.writer, "downloading source tarball to source.tar.gz")
		if err := c.downloadFile(u.String(), filepath.Join(destDir, "source.tar.gz")); err != nil {
			return InResponse{}, err
		}
	}

	if request.Params.IncludeSourceZip && foundRelease.TagName != nil {
		u, err := c.github.GetZipballLink(*foundRelease.TagName)
		if err != nil {
			return InResponse{}, err
		}
		fmt.Fprintln(c.writer, "downloading source zip to source.zip")
		if err := c.downloadFile(u.String(), filepath.Join(destDir, "source.zip")); err != nil {
			return InResponse{}, err
		}
	}

	return InResponse{
		Version:  versionFromRelease(foundRelease),
		Metadata: metadataFromRelease(foundRelease, commitSHA),
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file `%s`: HTTP status %d", filepath.Base(destPath), resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
