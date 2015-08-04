package resource_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/zachgersh/go-github/github"

	"github.com/concourse/github-release-resource"
	"github.com/concourse/github-release-resource/fakes"
)

var _ = Describe("In Command", func() {
	var (
		command      *resource.InCommand
		githubClient *fakes.FakeGitHub

		inRequest resource.InRequest

		inResponse resource.InResponse
		inErr      error

		tmpDir  string
		destDir string
	)

	BeforeEach(func() {
		var err error

		githubClient = &fakes.FakeGitHub{}
		command = resource.NewInCommand(githubClient, ioutil.Discard)

		tmpDir, err = ioutil.TempDir("", "github-release")
		Ω(err).ShouldNot(HaveOccurred())

		destDir = filepath.Join(tmpDir, "destination")

		githubClient.DownloadReleaseAssetReturns(ioutil.NopCloser(bytes.NewBufferString("some-content")), nil)

		inRequest = resource.InRequest{}
	})

	AfterEach(func() {
		Ω(os.RemoveAll(tmpDir)).Should(Succeed())
	})

	buildRelease := func(id int, tag string) *github.RepositoryRelease {
		return &github.RepositoryRelease{
			ID:      github.Int(id),
			TagName: github.String(tag),
			HTMLURL: github.String("http://google.com"),
			Name:    github.String("release-name"),
			Body:    github.String("*markdown*"),
		}
	}

	buildAsset := func(id int, name string) github.ReleaseAsset {
		return github.ReleaseAsset{
			ID:   github.Int(id),
			Name: &name,
		}
	}

	Context("when there is a tagged release", func() {
		Context("when a present version is specified", func() {
			BeforeEach(func() {
				githubClient.GetReleaseByTagReturns(buildRelease(1, "v0.35.0"), nil)

				githubClient.ListReleaseAssetsReturns([]github.ReleaseAsset{
					buildAsset(0, "example.txt"),
					buildAsset(1, "example.rtf"),
					buildAsset(2, "example.wtf"),
				}, nil)

				inRequest.Version = &resource.Version{
					Tag: "v0.35.0",
				}
			})

			Context("when valid asset filename globs are given", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{
						Globs: []string{"*.txt", "*.rtf"},
					}

					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("succeeds", func() {
					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("returns the fetched version", func() {
					Ω(inResponse.Version).Should(Equal(resource.Version{Tag: "v0.35.0"}))
				})

				It("has some sweet metadata", func() {
					Ω(inResponse.Metadata).Should(ConsistOf(
						resource.MetadataPair{Name: "url", Value: "http://google.com"},
						resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
						resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					))
				})

				PIt("downloads only the files that match the globs", func() {
				})
			})

			Context("when an invalid asset filename glob is given", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{
						Globs: []string{`[`},
					}

					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("returns an error", func() {
					Ω(inErr).Should(HaveOccurred())
				})
			})

			Context("when no globs are specified", func() {
				BeforeEach(func() {
					inRequest.Source = resource.Source{}
					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("succeeds", func() {
					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("returns the fetched version", func() {
					Ω(inResponse.Version).Should(Equal(resource.Version{Tag: "v0.35.0"}))
				})

				It("has some sweet metadata", func() {
					Ω(inResponse.Metadata).Should(ConsistOf(
						resource.MetadataPair{Name: "url", Value: "http://google.com"},
						resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
						resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					))
				})

				PIt("downloads all of the files", func() {
				})
			})

			Context("when downloading an asset fails", func() {
				BeforeEach(func() {
					githubClient.DownloadReleaseAssetReturns(nil, errors.New("not this time"))
					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("returns an error", func() {
					Ω(inErr).Should(HaveOccurred())
				})
			})

			Context("when listing release assets fails", func() {
				disaster := errors.New("nope")

				BeforeEach(func() {
					githubClient.ListReleaseAssetsReturns(nil, disaster)
					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("returns the error", func() {
					Ω(inErr).Should(Equal(disaster))
				})
			})
		})

		Context("when the version is not specified", func() {
			BeforeEach(func() {
				githubClient.LatestReleaseReturns(buildRelease(1, "v0.37.0"), nil)

				inRequest.Version = nil
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
				Ω(githubClient.GetReleaseByTagCallCount()).Should(Equal(0))
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(resource.Version{Tag: "v0.37.0"}))
			})

			It("has some sweet metadata", func() {
				Ω(inResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
				))
			})

			It("stores git tag in a file", func() {
				_, err := os.Stat(filepath.Join(destDir, "tag"))
				Ω(err).ShouldNot(HaveOccurred())

				tag, err := ioutil.ReadFile(filepath.Join(destDir, "tag"))
				Ω(err).ShouldNot(HaveOccurred())

				Ω(string(tag)).Should(Equal("v0.37.0"))
			})

			It("stores version in a file", func() {
				_, err := os.Stat(filepath.Join(destDir, "version"))
				Ω(err).ShouldNot(HaveOccurred())

				version, err := ioutil.ReadFile(filepath.Join(destDir, "version"))
				Ω(err).ShouldNot(HaveOccurred())

				Ω(string(version)).Should(Equal("0.37.0"))
			})

			PIt("fetches from the latest release", func() {
			})
		})
	})

	Context("when no tagged release is present", func() {
		BeforeEach(func() {
			githubClient.GetReleaseByTagReturns(nil, nil)

			inRequest.Version = &resource.Version{
				Tag: "v0.40.0",
			}

			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("returns an error", func() {
			Ω(inErr).Should(MatchError("no releases"))
		})
	})

	Context("when no latest release is present", func() {
		BeforeEach(func() {
			githubClient.LatestReleaseReturns(nil, nil)
			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("returns an error", func() {
			Ω(inErr).Should(MatchError("no releases"))
		})
	})

	Context("when getting a tagged release fails", func() {
		disaster := errors.New("nope")

		BeforeEach(func() {
			githubClient.GetReleaseByTagReturns(nil, disaster)

			inRequest.Version = &resource.Version{
				Tag: "some-tag",
			}
			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("returns the error", func() {
			Ω(inErr).Should(Equal(disaster))
		})
	})

	Context("when getting the latest release fails", func() {
		disaster := errors.New("nope-again")

		BeforeEach(func() {
			githubClient.LatestReleaseReturns(nil, disaster)
			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("returns the error", func() {
			Ω(inErr).Should(Equal(disaster))
		})
	})
})
