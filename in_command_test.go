package resource_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/google/go-github/v66/github"

	resource "github.com/concourse/github-release-resource"
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
		command = resource.NewInCommand(githubClient, io.Discard)

		tmpDir, err = os.MkdirTemp("", "github-release")
		Ω(err).ShouldNot(HaveOccurred())

		destDir = filepath.Join(tmpDir, "destination")

		githubClient.DownloadReleaseAssetReturns(io.NopCloser(bytes.NewBufferString("some-content")), nil)

		inRequest = resource.InRequest{}
	})

	AfterEach(func() {
		Ω(os.RemoveAll(tmpDir)).Should(Succeed())
	})

	buildRelease := func(id int64, tag string, draft bool) *github.RepositoryRelease {
		return &github.RepositoryRelease{
			ID:          github.Int64(id),
			TagName:     github.String(tag),
			HTMLURL:     github.String("http://google.com"),
			Name:        github.String("release-name"),
			Body:        github.String("*markdown*"),
			CreatedAt:   &github.Timestamp{Time: exampleTimeStamp(1)},
			PublishedAt: &github.Timestamp{Time: exampleTimeStamp(1)},
			Draft:       github.Bool(draft),
			Prerelease:  github.Bool(false),
		}
	}

	buildNilTagRelease := func(id int64) *github.RepositoryRelease {
		return &github.RepositoryRelease{
			ID:         github.Int64(id),
			HTMLURL:    github.String("http://google.com"),
			Name:       github.String("release-name"),
			Body:       github.String("*markdown*"),
			CreatedAt:  &github.Timestamp{Time: exampleTimeStamp(1)},
			Draft:      github.Bool(true),
			Prerelease: github.Bool(false),
		}
	}

	buildAsset := func(id int64, name string) *github.ReleaseAsset {
		state := "uploaded"
		return &github.ReleaseAsset{
			ID:    github.Int64(id),
			Name:  &name,
			State: &state,
		}
	}

	buildFailedAsset := func(id int64, name string) *github.ReleaseAsset {
		state := "starter"
		return &github.ReleaseAsset{
			ID:    github.Int64(id),
			Name:  &name,
			State: &state,
		}
	}

	Context("when there is a tagged release", func() {
		Context("when a present version is specified", func() {
			BeforeEach(func() {
				githubClient.GetReleaseReturns(buildRelease(1, "v0.35.0", false), nil)
				githubClient.ResolveTagToCommitSHAReturns("f28085a4a8f744da83411f5e09fd7b1709149eee", nil)

				githubClient.ListReleaseAssetsReturns([]*github.ReleaseAsset{
					buildAsset(0, "example.txt"),
					buildAsset(1, "example.rtf"),
					buildAsset(2, "example.wtf"),
					buildFailedAsset(3, "example.doc"),
				}, nil)

				inRequest.Version = &resource.Version{
					ID:  "1",
					Tag: "v0.35.0",
				}
			})

			Context("when valid asset filename globs are given", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{
						Globs: []string{"*.txt", "*.rtf"},
					}
				})

				It("succeeds", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					Ω(inErr).ShouldNot(HaveOccurred())
				})

				It("returns the fetched version", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					Ω(inResponse.Version).Should(Equal(newVersionWithTimestamp(1, "v0.35.0", 1)))
				})

				It("has some sweet metadata", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					Ω(inResponse.Metadata).Should(ConsistOf(
						resource.MetadataPair{Name: "url", Value: "http://google.com"},
						resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
						resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
						resource.MetadataPair{Name: "tag", Value: "v0.35.0"},
						resource.MetadataPair{Name: "commit_sha", Value: "f28085a4a8f744da83411f5e09fd7b1709149eee"},
					))
				})

				It("calls #GetReleast with the correct arguments", func() {
					command.Run(destDir, inRequest)

					Ω(githubClient.GetReleaseArgsForCall(0)).Should(Equal(1))
				})

				It("downloads only the files that match the globs", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					Expect(githubClient.DownloadReleaseAssetCallCount()).To(Equal(2))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(0)).Should(Equal(*buildAsset(0, "example.txt")))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(1)).Should(Equal(*buildAsset(1, "example.rtf")))
				})

				It("does create the body, tag, version, and url files", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					contents, err := os.ReadFile(path.Join(destDir, "tag"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("v0.35.0"))

					contents, err = os.ReadFile(path.Join(destDir, "version"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("0.35.0"))

					contents, err = os.ReadFile(path.Join(destDir, "commit_sha"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("f28085a4a8f744da83411f5e09fd7b1709149eee"))

					contents, err = os.ReadFile(path.Join(destDir, "body"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("*markdown*"))

					contents, err = os.ReadFile(path.Join(destDir, "timestamp"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("2018-01-01T00:00:00Z"))

					contents, err = os.ReadFile(path.Join(destDir, "url"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("http://google.com"))
				})

				Context("when there is a custom tag filter", func() {
					BeforeEach(func() {
						inRequest.Source = resource.Source{
							TagFilter: "package-(.*)",
						}
						githubClient.GetReleaseReturns(buildRelease(1, "package-0.35.0", false), nil)
						githubClient.ResolveTagToCommitSHAReturns("f28085a4a8f744da83411f5e09fd7b1709149eee", nil)
						inResponse, inErr = command.Run(destDir, inRequest)
					})

					It("succeeds", func() {
						inResponse, inErr = command.Run(destDir, inRequest)

						Expect(inErr).ToNot(HaveOccurred())
					})

					It("does create the tag, version, and url files", func() {
						inResponse, inErr = command.Run(destDir, inRequest)

						contents, err := os.ReadFile(path.Join(destDir, "tag"))
						Ω(err).ShouldNot(HaveOccurred())
						Ω(string(contents)).Should(Equal("package-0.35.0"))

						contents, err = os.ReadFile(path.Join(destDir, "version"))
						Ω(err).ShouldNot(HaveOccurred())
						Ω(string(contents)).Should(Equal("0.35.0"))

						contents, err = os.ReadFile(path.Join(destDir, "url"))
						Ω(err).ShouldNot(HaveOccurred())
						Ω(string(contents)).Should(Equal("http://google.com"))
					})
				})

				Context("when include_source_tarball is true", func() {
					var tarballUrl *url.URL

					BeforeEach(func() {
						inRequest.Params.IncludeSourceTarball = true

						tarballUrl, _ = url.Parse(githubServer.URL())
						tarballUrl.Path = "/gimme-a-tarball/"
					})

					Context("when getting the tarball link succeeds", func() {
						BeforeEach(func() {
							githubClient.GetTarballLinkReturns(tarballUrl, nil)
						})

						Context("when downloading the tarball succeeds", func() {
							BeforeEach(func() {
								githubServer.AppendHandlers(
									ghttp.CombineHandlers(
										ghttp.VerifyRequest("GET", tarballUrl.Path),
										ghttp.RespondWith(http.StatusOK, "source-tar-file-contents"),
									),
								)
							})

							It("succeeds", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								Expect(inErr).ToNot(HaveOccurred())
							})

							It("downloads the source tarball", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								Expect(githubServer.ReceivedRequests()).To(HaveLen(1))
							})

							It("saves the source tarball in the destination directory", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								fileContents, err := os.ReadFile(filepath.Join(destDir, "source.tar.gz"))
								fContents := string(fileContents)
								Expect(err).NotTo(HaveOccurred())
								Expect(fContents).To(Equal("source-tar-file-contents"))
							})

							It("saves the source tarball in the assets directory, if desired", func() {
								inRequest.Source.AssetDir = true
								inResponse, inErr = command.Run(destDir, inRequest)

								fileContents, err := os.ReadFile(filepath.Join(destDir, "assets", "source.tar.gz"))
								fContents := string(fileContents)
								Expect(err).NotTo(HaveOccurred())
								Expect(fContents).To(Equal("source-tar-file-contents"))
							})
						})

						Context("when downloading the tarball fails", func() {
							BeforeEach(func() {
								githubServer.AppendHandlers(
									ghttp.CombineHandlers(
										ghttp.VerifyRequest("GET", tarballUrl.Path),
										ghttp.RespondWith(http.StatusInternalServerError, ""),
									),
								)
							})

							It("returns an appropriate error", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								Expect(inErr).To(MatchError("failed to download file `source.tar.gz`: HTTP status 500"))
							})
						})
					})

					Context("when getting the tarball link fails", func() {
						disaster := errors.New("oh my")

						BeforeEach(func() {
							githubClient.GetTarballLinkReturns(nil, disaster)
						})

						It("returns the error", func() {
							inResponse, inErr = command.Run(destDir, inRequest)

							Expect(inErr).To(Equal(disaster))
						})
					})
				})

				Context("when include_source_zip is true", func() {
					var zipUrl *url.URL

					BeforeEach(func() {
						inRequest.Params.IncludeSourceZip = true

						zipUrl, _ = url.Parse(githubServer.URL())
						zipUrl.Path = "/gimme-a-zip/"
					})

					Context("when getting the zip link succeeds", func() {
						BeforeEach(func() {
							githubClient.GetZipballLinkReturns(zipUrl, nil)
						})

						Context("when downloading the zip succeeds", func() {
							BeforeEach(func() {
								githubServer.AppendHandlers(
									ghttp.CombineHandlers(
										ghttp.VerifyRequest("GET", zipUrl.Path),
										ghttp.RespondWith(http.StatusOK, "source-zip-file-contents"),
									),
								)
							})

							It("succeeds", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								Expect(inErr).ToNot(HaveOccurred())
							})

							It("downloads the source zip", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								Expect(githubServer.ReceivedRequests()).To(HaveLen(1))
							})

							It("saves the source zip in the destination directory", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								fileContents, err := os.ReadFile(filepath.Join(destDir, "source.zip"))
								fContents := string(fileContents)
								Expect(err).NotTo(HaveOccurred())
								Expect(fContents).To(Equal("source-zip-file-contents"))
							})

							It("saves the source tarball in the assets directory, if desired", func() {
								inRequest.Source.AssetDir = true
								inResponse, inErr = command.Run(destDir, inRequest)

								fileContents, err := os.ReadFile(filepath.Join(destDir, "assets", "source.zip"))
								fContents := string(fileContents)
								Expect(err).NotTo(HaveOccurred())
								Expect(fContents).To(Equal("source-zip-file-contents"))
							})
						})

						Context("when downloading the zip fails", func() {
							BeforeEach(func() {
								githubServer.AppendHandlers(
									ghttp.CombineHandlers(
										ghttp.VerifyRequest("GET", zipUrl.Path),
										ghttp.RespondWith(http.StatusInternalServerError, ""),
									),
								)
							})

							It("returns an appropriate error", func() {
								inResponse, inErr = command.Run(destDir, inRequest)

								Expect(inErr).To(MatchError("failed to download file `source.zip`: HTTP status 500"))
							})
						})
					})

					Context("when getting the zip link fails", func() {
						disaster := errors.New("oh my")

						BeforeEach(func() {
							githubClient.GetZipballLinkReturns(nil, disaster)
						})

						It("returns the error", func() {
							inResponse, inErr = command.Run(destDir, inRequest)

							Expect(inErr).To(Equal(disaster))
						})
					})
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
					Ω(inResponse.Version).Should(Equal(newVersionWithTimestamp(1, "v0.35.0", 1)))
				})

				It("has some sweet metadata", func() {
					Ω(inResponse.Metadata).Should(ConsistOf(
						resource.MetadataPair{Name: "url", Value: "http://google.com"},
						resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
						resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
						resource.MetadataPair{Name: "tag", Value: "v0.35.0"},
						resource.MetadataPair{Name: "commit_sha", Value: "f28085a4a8f744da83411f5e09fd7b1709149eee"},
					))
				})

				It("downloads all of the files", func() {
					Ω(githubClient.DownloadReleaseAssetArgsForCall(0)).Should(Equal(*buildAsset(0, "example.txt")))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(1)).Should(Equal(*buildAsset(1, "example.rtf")))
					Ω(githubClient.DownloadReleaseAssetArgsForCall(2)).Should(Equal(*buildAsset(2, "example.wtf")))
					Ω(githubClient.DownloadReleaseAssetCallCount()).Should(Equal(3))
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
			githubClient.GetReleaseReturns(nil, nil)

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
		disaster := errors.New("no releases")

		BeforeEach(func() {
			githubClient.GetReleaseReturns(nil, disaster)

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
				githubClient.GetReleaseReturns(buildRelease(1, "v0.35.0", true), nil)

				inRequest.Version = &resource.Version{ID: "1"}
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(newVersionWithTimestamp(1, "v0.35.0", 1)))
			})

			It("has some sweet metadata", func() {
				Ω(inResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "v0.35.0"},
					resource.MetadataPair{Name: "draft", Value: "true"},
				))
			})

			It("does create the tag, version, and URL files", func() {
				contents, err := os.ReadFile(path.Join(destDir, "tag"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("v0.35.0"))

				contents, err = os.ReadFile(path.Join(destDir, "version"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("0.35.0"))

				contents, err = os.ReadFile(path.Join(destDir, "url"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("http://google.com"))
			})
		})

		Context("which has an empty tag", func() {
			BeforeEach(func() {
				githubClient.GetReleaseReturns(buildRelease(1, "", true), nil)

				inRequest.Version = &resource.Version{ID: "1"}
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(newVersionWithTimestamp(1, "", 1)))
			})

			It("has some sweet metadata", func() {
				Ω(inResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: ""},
					resource.MetadataPair{Name: "draft", Value: "true"},
				))
			})

			It("does not create the tag and version files", func() {
				Ω(path.Join(destDir, "tag")).ShouldNot(BeAnExistingFile())
				Ω(path.Join(destDir, "version")).ShouldNot(BeAnExistingFile())
				Ω(path.Join(destDir, "commit_sha")).ShouldNot(BeAnExistingFile())
			})

			It("does create the url file", func() {
				contents, err := os.ReadFile(path.Join(destDir, "url"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("http://google.com"))
			})
		})

		Context("which has a nil tag", func() {
			BeforeEach(func() {
				githubClient.GetReleaseReturns(buildNilTagRelease(1), nil)

				inRequest.Version = &resource.Version{ID: "1"}
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(newVersionWithTimestamp(1, "", 1)))
			})

			It("has some sweet metadata", func() {
				Ω(inResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "draft", Value: "true"},
				))
			})

			It("does not create the tag and version files", func() {
				Ω(path.Join(destDir, "tag")).ShouldNot(BeAnExistingFile())
				Ω(path.Join(destDir, "version")).ShouldNot(BeAnExistingFile())
				Ω(path.Join(destDir, "commit_sha")).ShouldNot(BeAnExistingFile())
			})

			It("does create the url file", func() {
				contents, err := os.ReadFile(path.Join(destDir, "url"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("http://google.com"))
			})
		})
	})
})
