package resource_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/google/go-github/github"

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

	buildRelease := func(id int, tag string, draft bool, prerelease bool) *github.RepositoryRelease {
		return &github.RepositoryRelease{
			ID:         github.Int(id),
			TagName:    github.String(tag),
			HTMLURL:    github.String("http://google.com"),
			Name:       github.String("release-name"),
			Body:       github.String("*markdown*"),
			Draft:      github.Bool(draft),
			Prerelease: github.Bool(prerelease),
		}
	}

	buildNilTagRelease := func(id int) *github.RepositoryRelease {
		return &github.RepositoryRelease{
			ID:         github.Int(id),
			HTMLURL:    github.String("http://google.com"),
			Name:       github.String("release-name"),
			Body:       github.String("*markdown*"),
			Draft:      github.Bool(true),
			Prerelease: github.Bool(false),
		}
	}

	buildAsset := func(id int, name string) *github.ReleaseAsset {
		return &github.ReleaseAsset{
			ID:   github.Int(id),
			Name: &name,
		}
	}

	buildTagRef := func(tagRef, commitSHA string) *github.Reference {
		return &github.Reference{
			Ref: github.String(tagRef),
			URL: github.String("https://example.com"),
			Object: &github.GitObject{
				Type: github.String("commit"),
				SHA:  github.String(commitSHA),
				URL:  github.String("https://example.com"),
			},
		}
	}

	Context("when there is a tagged release", func() {
		Context("when a present version is specified", func() {
			BeforeEach(func() {
				githubClient.GetReleaseReturns(buildRelease(1, "v0.35.0", false, false), nil)
				githubClient.GetRefReturns(buildTagRef("v0.35.0", "f28085a4a8f744da83411f5e09fd7b1709149eee"), nil)

				githubClient.ListReleaseAssetsReturns([]*github.ReleaseAsset{
					buildAsset(0, "example.txt"),
					buildAsset(1, "example.rtf"),
					buildAsset(2, "example.wtf"),
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

					Ω(inResponse.Version).Should(Equal(resource.Version{ID: "1", Tag: "v0.35.0"}))
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

				It("does create the metadata files", func() {
					inResponse, inErr = command.Run(destDir, inRequest)

					contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("v0.35.0"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("0.35.0"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "commit_sha"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("f28085a4a8f744da83411f5e09fd7b1709149eee"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "body"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("*markdown*"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "draft"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("0"))

					contents, err = ioutil.ReadFile(path.Join(destDir, "prerelease"))
					Ω(err).ShouldNot(HaveOccurred())
					Ω(string(contents)).Should(Equal("0"))
				})

				Context("when there is a custom tag filter", func() {
					BeforeEach(func() {
						inRequest.Source = resource.Source{
							TagFilter: "package-(.*)",
						}
						githubClient.GetReleaseReturns(buildRelease(1, "package-0.35.0", false, false), nil)
						githubClient.GetRefReturns(buildTagRef("package-0.35.0", "f28085a4a8f744da83411f5e09fd7b1709149eee"), nil)
						inResponse, inErr = command.Run(destDir, inRequest)
					})

					It("succeeds", func() {
						inResponse, inErr = command.Run(destDir, inRequest)

						Expect(inErr).ToNot(HaveOccurred())
					})

					It("does create the metadata files", func() {
						inResponse, inErr = command.Run(destDir, inRequest)

						contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
						Ω(err).ShouldNot(HaveOccurred())
						Ω(string(contents)).Should(Equal("package-0.35.0"))

						contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
						Ω(err).ShouldNot(HaveOccurred())
						Ω(string(contents)).Should(Equal("0.35.0"))
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

								fileContents, err := ioutil.ReadFile(filepath.Join(destDir, "source.tar.gz"))
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

								fileContents, err := ioutil.ReadFile(filepath.Join(destDir, "source.zip"))
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
					Ω(inResponse.Version).Should(Equal(resource.Version{ID: "1", Tag: "v0.35.0"}))
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
		disaster := errors.New("nope")

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

	Context("when there is a pre-release", func() {
		BeforeEach(func() {
			githubClient.GetReleaseReturns(buildRelease(1, "v0.35.0", false, true), nil)
			githubClient.GetRefReturns(buildTagRef("v0.35.0", "f28085a4a8f744da83411f5e09fd7b1709149eee"), nil)

			inRequest.Version = &resource.Version{ID: "1"}
			inResponse, inErr = command.Run(destDir, inRequest)
		})

		It("succeeds", func() {
			Ω(inErr).ShouldNot(HaveOccurred())
		})

		It("returns the fetched version", func() {
			Ω(inResponse.Version).Should(Equal(resource.Version{ID: "1", Tag: "v0.35.0"}))
		})

		It("has some sweet metadata", func() {
			Ω(inResponse.Metadata).Should(ConsistOf(
				resource.MetadataPair{Name: "url", Value: "http://google.com"},
				resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
				resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
				resource.MetadataPair{Name: "tag", Value: "v0.35.0"},
				resource.MetadataPair{Name: "pre-release", Value: "true"},
				resource.MetadataPair{Name: "commit_sha", Value: "f28085a4a8f744da83411f5e09fd7b1709149eee"},
			))
		})

		It("does create the metadata files", func() {
			contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(contents)).Should(Equal("v0.35.0"))

			contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(contents)).Should(Equal("0.35.0"))

			contents, err = ioutil.ReadFile(path.Join(destDir, "draft"))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(contents)).Should(Equal("0"))

			contents, err = ioutil.ReadFile(path.Join(destDir, "prerelease"))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(contents)).Should(Equal("1"))
		})
	})

	Context("when there is a draft release", func() {
		Context("which has a tag", func() {
			BeforeEach(func() {
				githubClient.GetReleaseReturns(buildRelease(1, "v0.35.0", true, false), nil)

				inRequest.Version = &resource.Version{ID: "1"}
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(resource.Version{ID: "1", Tag: "v0.35.0"}))
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

			It("does create the metadata files", func() {
				contents, err := ioutil.ReadFile(path.Join(destDir, "tag"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("v0.35.0"))

				contents, err = ioutil.ReadFile(path.Join(destDir, "version"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("0.35.0"))

				contents, err = ioutil.ReadFile(path.Join(destDir, "draft"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("1"))

				contents, err = ioutil.ReadFile(path.Join(destDir, "prerelease"))
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(contents)).Should(Equal("0"))
			})
		})

		Context("which has an empty tag", func() {
			BeforeEach(func() {
				githubClient.GetReleaseReturns(buildRelease(1, "", true, false), nil)

				inRequest.Version = &resource.Version{ID: "1"}
				inResponse, inErr = command.Run(destDir, inRequest)
			})

			It("succeeds", func() {
				Ω(inErr).ShouldNot(HaveOccurred())
			})

			It("returns the fetched version", func() {
				Ω(inResponse.Version).Should(Equal(resource.Version{ID: "1"}))
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
				Ω(inResponse.Version).Should(Equal(resource.Version{ID: "1"}))
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
		})
	})
})
