package resource_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	resource "github.com/concourse/github-release-resource"
	"github.com/concourse/github-release-resource/fakes"

	"github.com/google/go-github/v66/github"
)

func file(path, contents string) {
	Ω(os.WriteFile(path, []byte(contents), 0644)).Should(Succeed())
}

var _ = Describe("Out Command", func() {
	var (
		command      *resource.OutCommand
		githubClient *fakes.FakeGitHub

		sourcesDir string

		request resource.OutRequest
	)

	BeforeEach(func() {
		var err error

		githubClient = &fakes.FakeGitHub{}
		command = resource.NewOutCommand(githubClient, io.Discard)

		sourcesDir, err = os.MkdirTemp("", "github-release")
		Ω(err).ShouldNot(HaveOccurred())

		githubClient.CreateReleaseStub = func(gh github.RepositoryRelease) (*github.RepositoryRelease, error) {
			createdRel := gh
			createdRel.ID = github.Int64(112)
			createdRel.HTMLURL = github.String("http://google.com")
			createdRel.Name = github.String("release-name")
			createdRel.Body = github.String("*markdown*")
			return &createdRel, nil
		}

		githubClient.UpdateReleaseStub = func(gh github.RepositoryRelease) (*github.RepositoryRelease, error) {
			return &gh, nil
		}
	})

	AfterEach(func() {
		Ω(os.RemoveAll(sourcesDir)).Should(Succeed())
	})

	Context("when the release has already been created", func() {
		existingAssets := []github.ReleaseAsset{
			{
				ID:   github.Int64(456789),
				Name: github.String("unicorns.txt"),
			},
			{
				ID:    github.Int64(3450798),
				Name:  github.String("rainbows.txt"),
				State: github.String("new"),
			},
		}

		existingReleases := []github.RepositoryRelease{
			{
				ID:    github.Int64(1),
				Draft: github.Bool(true),
			},
			{
				ID:      github.Int64(112),
				TagName: github.String("some-tag-name"),
				Assets:  []*github.ReleaseAsset{&existingAssets[0]},
				Draft:   github.Bool(false),
			},
		}

		BeforeEach(func() {
			githubClient.ListReleasesStub = func() ([]*github.RepositoryRelease, error) {
				var rels []*github.RepositoryRelease
				for _, r := range existingReleases {
					c := r
					rels = append(rels, &c)
				}

				return rels, nil
			}

			githubClient.ListReleaseAssetsStub = func(github.RepositoryRelease) ([]*github.ReleaseAsset, error) {
				var assets []*github.ReleaseAsset
				for _, a := range existingAssets {
					c := a
					assets = append(assets, &c)
				}

				return assets, nil
			}

			namePath := filepath.Join(sourcesDir, "name")
			bodyPath := filepath.Join(sourcesDir, "body")
			tagPath := filepath.Join(sourcesDir, "tag")

			file(namePath, "v0.3.12")
			file(bodyPath, "this is a great release")
			file(tagPath, "some-tag-name")

			request = resource.OutRequest{
				Params: resource.OutParams{
					NamePath: "name",
					BodyPath: "body",
					TagPath:  "tag",
				},
			}
		})

		It("deletes the existing assets", func() {
			_, err := command.Run(sourcesDir, request)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(githubClient.ListReleaseAssetsCallCount()).Should(Equal(1))
			Ω(githubClient.ListReleaseAssetsArgsForCall(0)).Should(Equal(existingReleases[1]))

			Ω(githubClient.DeleteReleaseAssetCallCount()).Should(Equal(2))

			Ω(githubClient.DeleteReleaseAssetArgsForCall(0)).Should(Equal(existingAssets[0]))
			Ω(githubClient.DeleteReleaseAssetArgsForCall(1)).Should(Equal(existingAssets[1]))
		})

		Context("when not set as a draft release", func() {
			BeforeEach(func() {
				request.Source.Drafts = false
			})

			It("updates the existing release to a non-draft", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := githubClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Draft).Should(Equal(false))
			})
		})

		Context("when set as a draft release", func() {
			BeforeEach(func() {
				request.Source.Drafts = true
			})

			It("updates the existing release to a draft", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := githubClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Draft).Should(Equal(true))
			})
		})

		Context("when a body is not supplied", func() {
			BeforeEach(func() {
				request.Params.BodyPath = ""
			})

			It("does not blow away the body", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := githubClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(updatedRelease.Body).Should(BeNil())
			})
		})

		Context("when a commitish is not supplied", func() {
			It("updates the existing release", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := githubClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Body).Should(Equal("this is a great release"))
				Ω(updatedRelease.TargetCommitish).Should(Equal(github.String("")))
			})
		})

		Context("when a commitish is supplied", func() {
			BeforeEach(func() {
				commitishPath := filepath.Join(sourcesDir, "commitish")
				file(commitishPath, "1z22f1")
				request.Params.CommitishPath = "commitish"
			})

			It("updates the existing release", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := githubClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Body).Should(Equal("this is a great release"))
				Ω(updatedRelease.TargetCommitish).Should(Equal(github.String("1z22f1")))
			})
		})

		Context("when set to autogenerate release notes", func() {
			BeforeEach(func() {
				request.Params.GenerateReleaseNotes = true
			})
			// See https://github.com/google/go-github/issues/2444
			It("has no effect on updating the existing release", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := githubClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Body).Should(Equal("this is a great release"))
				Ω(updatedRelease.GenerateReleaseNotes).Should(BeNil())
			})
		})
	})

	Context("when the release has not already been created", func() {
		BeforeEach(func() {
			namePath := filepath.Join(sourcesDir, "name")
			tagPath := filepath.Join(sourcesDir, "tag")

			file(namePath, "v0.3.12")
			file(tagPath, "0.3.12")

			request = resource.OutRequest{
				Params: resource.OutParams{
					NamePath: "name",
					TagPath:  "tag",
				},
			}
		})

		Context("with a commitish", func() {
			BeforeEach(func() {
				commitishPath := filepath.Join(sourcesDir, "commitish")
				file(commitishPath, "a2f4a3")
				request.Params.CommitishPath = "commitish"
			})

			It("creates a release on GitHub with the commitish", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(release.TargetCommitish).Should(Equal(github.String("a2f4a3")))
			})
		})

		Context("without a commitish", func() {
			It("creates a release on GitHub without the commitish", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				// GitHub treats empty string the same as not suppying the field.
				Ω(release.TargetCommitish).Should(Equal(github.String("")))
			})
		})

		Context("with a body", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Params.BodyPath = "body"
			})

			It("creates a release on GitHub", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal("this is a great release"))
			})
		})

		Context("without a body", func() {
			It("works", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
			})
		})

		It("always defaults to non-draft mode", func() {
			_, err := command.Run(sourcesDir, request)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
			release := githubClient.CreateReleaseArgsForCall(0)

			Ω(*release.Draft).Should(Equal(false))
		})

		Context("when pre-release are set and release are not", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Source.Release = false
				request.Source.PreRelease = true
			})

			It("creates a non-draft pre-release in Github", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
				Ω(*release.Draft).Should(Equal(false))
				Ω(*release.Prerelease).Should(Equal(true))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
					resource.MetadataPair{Name: "pre-release", Value: "true"},
				))
			})
		})

		Context("when release and pre-release are set", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Source.Release = true
				request.Source.PreRelease = true
			})

			It("creates a final release in Github", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
				Ω(*release.Draft).Should(Equal(false))
				Ω(*release.Prerelease).Should(Equal(false))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
				))
			})
		})

		Context("when set as a draft release", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Source.Drafts = true
			})

			It("creates a release on GitHub in draft mode", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
				Ω(*release.Draft).Should(Equal(true))
				Ω(*release.Prerelease).Should(Equal(false))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
					resource.MetadataPair{Name: "draft", Value: "true"},
				))
			})
		})

		Context("with file globs", func() {
			BeforeEach(func() {
				globMatching := filepath.Join(sourcesDir, "great-file.tgz")
				globNotMatching := filepath.Join(sourcesDir, "bad-file.txt")

				file(globMatching, "matching")
				file(globNotMatching, "not matching")

				request = resource.OutRequest{
					Params: resource.OutParams{
						NamePath: "name",
						BodyPath: "body",
						TagPath:  "tag",

						Globs: []string{
							"*.tgz",
						},
					},
				}

				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "*markdown*")
				request.Params.BodyPath = "body"
			})

			It("uploads matching file globs", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.UploadReleaseAssetCallCount()).Should(Equal(1))
				release, name, file := githubClient.UploadReleaseAssetArgsForCall(0)

				Ω(*release.ID).Should(Equal(int64(112)))
				Ω(name).Should(Equal("great-file.tgz"))
				Ω(file.Name()).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))
			})

			It("has some sweet metadata", func() {
				outResponse, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(outResponse.Metadata).Should(ConsistOf(
					resource.MetadataPair{Name: "url", Value: "http://google.com"},
					resource.MetadataPair{Name: "name", Value: "release-name", URL: "http://google.com"},
					resource.MetadataPair{Name: "body", Value: "*markdown*", Markdown: true},
					resource.MetadataPair{Name: "tag", Value: "0.3.12"},
				))
			})

			It("returns an error if a glob is provided that does not match any files", func() {
				request.Params.Globs = []string{
					"*.tgz",
					"*.gif",
				}

				_, err := command.Run(sourcesDir, request)
				Ω(err).Should(HaveOccurred())
				Ω(err).Should(MatchError("could not find file that matches glob '*.gif'"))
			})

			Context("when upload release asset fails", func() {
				BeforeEach(func() {
					existingAsset := false
					githubClient.DeleteReleaseAssetStub = func(github.ReleaseAsset) error {
						existingAsset = false
						return nil
					}

					githubClient.ListReleaseAssetsReturns([]*github.ReleaseAsset{
						{
							ID:   github.Int64(456789),
							Name: github.String("great-file.tgz"),
						},
						{
							ID:   github.Int64(3450798),
							Name: github.String("whatever.tgz"),
						},
					}, nil)

					githubClient.UploadReleaseAssetStub = func(rel github.RepositoryRelease, name string, file *os.File) error {
						Expect(io.ReadAll(file)).To(Equal([]byte("matching")))
						Expect(existingAsset).To(BeFalse())
						existingAsset = true
						return errors.New("some-error")
					}
				})

				It("retries 10 times", func() {
					_, err := command.Run(sourcesDir, request)
					Expect(err).To(Equal(errors.New("some-error")))

					Ω(githubClient.UploadReleaseAssetCallCount()).Should(Equal(10))
					Ω(githubClient.ListReleaseAssetsCallCount()).Should(Equal(10))
					Ω(*githubClient.ListReleaseAssetsArgsForCall(9).ID).Should(Equal(int64(112)))

					actualRelease, actualName, actualFile := githubClient.UploadReleaseAssetArgsForCall(9)
					Ω(*actualRelease.ID).Should(Equal(int64(112)))
					Ω(actualName).Should(Equal("great-file.tgz"))
					Ω(actualFile.Name()).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))

					Ω(githubClient.DeleteReleaseAssetCallCount()).Should(Equal(10))
					actualAsset := githubClient.DeleteReleaseAssetArgsForCall(8)
					Expect(*actualAsset.ID).To(Equal(int64(456789)))
				})

				Context("when uploading succeeds on the 5th attempt", func() {
					BeforeEach(func() {
						results := make(chan error, 6)
						results <- errors.New("1")
						results <- errors.New("2")
						results <- errors.New("3")
						results <- errors.New("4")
						results <- nil
						results <- errors.New("6")

						githubClient.UploadReleaseAssetStub = func(github.RepositoryRelease, string, *os.File) error {
							return <-results
						}
					})

					It("succeeds", func() {
						_, err := command.Run(sourcesDir, request)
						Expect(err).ToNot(HaveOccurred())

						Ω(githubClient.UploadReleaseAssetCallCount()).Should(Equal(5))
						Ω(githubClient.ListReleaseAssetsCallCount()).Should(Equal(4))
						Ω(*githubClient.ListReleaseAssetsArgsForCall(3).ID).Should(Equal(int64(112)))

						actualRelease, actualName, actualFile := githubClient.UploadReleaseAssetArgsForCall(4)
						Ω(*actualRelease.ID).Should(Equal(int64(112)))
						Ω(actualName).Should(Equal("great-file.tgz"))
						Ω(actualFile.Name()).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))

						Ω(githubClient.DeleteReleaseAssetCallCount()).Should(Equal(4))
						actualAsset := githubClient.DeleteReleaseAssetArgsForCall(3)
						Expect(*actualAsset.ID).To(Equal(int64(456789)))
					})
				})
			})
		})

		Context("when the tag_prefix is set", func() {
			BeforeEach(func() {
				namePath := filepath.Join(sourcesDir, "name")
				tagPath := filepath.Join(sourcesDir, "tag")

				file(namePath, "v0.3.12")
				file(tagPath, "0.3.12")

				request = resource.OutRequest{
					Params: resource.OutParams{
						NamePath:  "name",
						TagPath:   "tag",
						TagPrefix: "version-",
					},
				}
			})

			It("appends the TagPrefix onto the TagName", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("version-0.3.12"))
			})
		})

		Context("with generate_release_notes set to false", func() {
			BeforeEach(func() {
				request.Params.GenerateReleaseNotes = false
			})

			It("creates a release on GitHub without autogenerated release notes", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(release.GenerateReleaseNotes).Should(Equal(github.Bool(false)))
			})
		})

		Context("with generate_release_notes set to true", func() {
			BeforeEach(func() {
				request.Params.GenerateReleaseNotes = true
			})

			It("creates a release on GitHub with autogenerated release notes", func() {
				_, err := command.Run(sourcesDir, request)
				Ω(err).ShouldNot(HaveOccurred())

				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(release.GenerateReleaseNotes).Should(Equal(github.Bool(true)))
			})
		})
	})
})
