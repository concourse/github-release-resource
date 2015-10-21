package resource_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/zachgersh/go-github/github"

	"github.com/concourse/github-release-resource"
	"github.com/concourse/github-release-resource/fakes"
)

var _ = Describe("In Command", func() {
	var (
		command      *resource.InCommand
		githubClient *fakes.FakeGitHub
		githubServer *ghttp.Server

		inRequest resource.InRequest

		inResponse resource.InResponse
		inErr      error

		tmpDir  string
		destDir string
	)

	BeforeEach(func() {
		var err error

		githubClient = &fakes.FakeGitHub{}
		githubServer = ghttp.NewServer()
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

	buildRelease := func(id int, tag string, draft bool) *github.RepositoryRelease {
		return &github.RepositoryRelease{
			ID:      github.Int(id),
			TagName: github.String(tag),
			HTMLURL: github.String("http://google.com"),
			Name:    github.String("release-name"),
			Body:    github.String("*markdown*"),
			Draft:   github.Bool(draft),
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
				githubClient.GetReleaseByTagReturns(buildRelease(1, "v0.35.0", false), nil)

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

				It("downloads only the files that match the globs", func() {
					Expect(githubClient.DownloadReleaseAssetCallCount()).To(Equal(2))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(0)).Should(Equal(buildAsset(0, "example.txt")))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(1)).Should(Equal(buildAsset(1, "example.rtf")))
				})

				It("does create the tag and version files", func() {
					contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("v0.35.0"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("0.35.0"))
				})
			})

			Context("when valid asset filename globs are given and include_source_tarball is true", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{
						Globs: []string{"*.txt", "*.rtf"},
					}
					inRequest.Params.IncludeSourceTarball = true

					tarballUrl, _ := url.Parse(githubServer.URL())
					tarballUrl.Path = "/gimme-a-tarball/"
					githubClient.GetTarballLinkReturns(tarballUrl, nil)
					githubServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", tarballUrl.Path),
							ghttp.RespondWith(200, "source-tar-file-contents"),
						),
					)

					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("succeeds", func() {
					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("downloads only the files that match the globs", func() {
					Expect(githubClient.DownloadReleaseAssetCallCount()).To(Equal(2))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(0)).Should(Equal(buildAsset(0, "example.txt")))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(1)).Should(Equal(buildAsset(1, "example.rtf")))
				})

				It("downloads the source tarball", func() {
					Expect(githubServer.ReceivedRequests()).To(HaveLen(1))
				})

				It("saves the source tarball in the destination directory", func() {
					fileContents, err := ioutil.ReadFile(filepath.Join(destDir, "source.tar.gz"))
					fContents := string(fileContents)
					Expect(err).NotTo(HaveOccurred())
					Expect(fContents).To(Equal("source-tar-file-contents"))
				})
			})

			Context("when include_source_tarball is true and no globs are specified", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{}
					inRequest.Params.IncludeSourceTarball = true

					tarballUrl, _ := url.Parse(githubServer.URL())
					tarballUrl.Path = "/gimme-a-tarball/"
					githubClient.GetTarballLinkReturns(tarballUrl, nil)
					githubServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", tarballUrl.Path),
							ghttp.RespondWith(200, "source-tar-file-contents"),
						),
					)

					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("succeeds", func() {
					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("downloads all the assets", func() {
					Expect(githubClient.DownloadReleaseAssetCallCount()).To(Equal(3))
					Expect(githubClient.DownloadReleaseAssetArgsForCall(0)).To(Equal(buildAsset(0, "example.txt")))
					Expect(githubClient.DownloadReleaseAssetArgsForCall(1)).To(Equal(buildAsset(1, "example.rtf")))
					Expect(githubClient.DownloadReleaseAssetArgsForCall(2)).To(Equal(buildAsset(2, "example.wtf")))
				})

				It("downloads the source tarball", func() {
					Expect(githubServer.ReceivedRequests()).To(HaveLen(1))
				})

				It("saves the source tarball in the destination directory", func() {
					fileContents, err := ioutil.ReadFile(filepath.Join(destDir, "source.tar.gz"))
					Expect(err).NotTo(HaveOccurred())
					Expect(string(fileContents)).To(Equal("source-tar-file-contents"))
				})
			})

			Context("when include_source_zip is true and no globs are specified", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{}
					inRequest.Params.IncludeSourceZip = true

					tarballUrl, _ := url.Parse(githubServer.URL())
					tarballUrl.Path = "/gimme-a-zip/"
					githubClient.GetZipballLinkReturns(tarballUrl, nil)
					githubServer.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", tarballUrl.Path),
							ghttp.RespondWith(200, "source-zip-file-contents"),
						),
					)

					inResponse, inErr = command.Run(destDir, inRequest)
				})

				It("succeeds", func() {
					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("downloads all the assets", func() {
					Expect(githubClient.DownloadReleaseAssetCallCount()).To(Equal(3))
					Expect(githubClient.DownloadReleaseAssetArgsForCall(0)).To(Equal(buildAsset(0, "example.txt")))
					Expect(githubClient.DownloadReleaseAssetArgsForCall(1)).To(Equal(buildAsset(1, "example.rtf")))
					Expect(githubClient.DownloadReleaseAssetArgsForCall(2)).To(Equal(buildAsset(2, "example.wtf")))
				})

				It("downloads the source zip", func() {
					Expect(githubServer.ReceivedRequests()).To(HaveLen(1))
				})

				It("saves the source zip in the destination directory", func() {
					fileContents, err := ioutil.ReadFile(filepath.Join(destDir, "source.zip"))
					Expect(err).NotTo(HaveOccurred())
					Expect(string(fileContents)).To(Equal("source-zip-file-contents"))
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

				It("downloads all of the files", func() {
					Ω(githubClient.DownloadReleaseAssetArgsForCall(0)).Should(Equal(buildAsset(0, "example.txt")))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(1)).Should(Equal(buildAsset(1, "example.rtf")))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(2)).Should(Equal(buildAsset(2, "example.wtf")))
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

	Context("when there is a draft release", func() {
		Context("which has a tag", func() {
			BeforeEach(func() {
				githubClient.GetReleaseByTagReturns(buildRelease(1, "v0.35.0", true), nil)

				inRequest.Version = &resource.Version{Tag: "v0.35.0"}
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

			It("does create the tag and version files", func() {
				contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("v0.35.0"))

				contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("0.35.0"))
			})
		})

		Context("which doesn't have a tag", func() {
			BeforeEach(func() {
				githubClient.GetReleaseByTagReturns(buildRelease(1, "", true), nil)

				inRequest.Version = &resource.Version{}
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(resource.Version{Tag: ""}))
			})

			It("has some sweet metadata", func() {
				Ω(inResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
				))
			})

			It("does not create the tag and version files", func() {
				Ω(path.Join(destDir, "tag")).ShouldNot(BeAnExistingFile())
				Ω(path.Join(destDir, "version")).ShouldNot(BeAnExistingFile())
			})
		})
	})
})
