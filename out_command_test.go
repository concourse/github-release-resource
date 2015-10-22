package resource_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/concourse/github-release-resource"
	"github.com/concourse/github-release-resource/fakes"

	"github.com/zachgersh/go-github/github"
)

func file(path, contents string) {
	Ω(ioutil.WriteFile(path, []byte(contents), 0644)).Should(Succeed())
}

var _ = Describe("Out Command", func() {
	var (
		command      *resource.OutCommand
		githubClient *fakes.FakeGitHub

		sourcesDir string

		request  resource.OutRequest
		response resource.OutResponse
	)

	BeforeEach(func() {
		var err error

		githubClient = &fakes.FakeGitHub{}
		command = resource.NewOutCommand(githubClient, ioutil.Discard)

		sourcesDir, err = ioutil.TempDir("", "github-release")
		Ω(err).ShouldNot(HaveOccurred())

		githubClient.CreateReleaseStub = func(gh github.RepositoryRelease) (*github.RepositoryRelease, error) {
			createdRel := gh
			createdRel.ID = github.Int(112)
			createdRel.HTMLURL = github.String("http://google.com")
			createdRel.Name = github.String("release-name")
			createdRel.Body = github.String("*markdown*")
			return &createdRel, nil
		}

		githubClient.UpdateReleaseStub = func(gh github.RepositoryRelease) (*github.RepositoryRelease, error) {
			return &gh, nil
		}
	})

	JustBeforeEach(func() {
		var err error
		response, err = command.Run(sourcesDir, request)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Ω(os.RemoveAll(sourcesDir)).Should(Succeed())
	})

	Context("when the release has already been created", func() {
		existingAssets := []github.ReleaseAsset{
			{
				ID:   github.Int(456789),
				Name: github.String("unicorns.txt"),
			},
			{
				ID:   github.Int(3450798),
				Name: github.String("rainbows.txt"),
			},
		}

		BeforeEach(func() {
			githubClient.ListReleasesReturns([]github.RepositoryRelease{
				{
					ID:    github.Int(1),
					Draft: github.Bool(true),
				},
				{
					ID:      github.Int(112),
					TagName: github.String("some-tag-name"),
					Assets:  existingAssets,
					Draft:   github.Bool(false),
				},
			}, nil)

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
			Ω(githubClient.DeleteReleaseAssetCallCount()).Should(Equal(2))

			Ω(githubClient.DeleteReleaseAssetArgsForCall(0)).Should(Equal(existingAssets[0]))
			Ω(githubClient.DeleteReleaseAssetArgsForCall(1)).Should(Equal(existingAssets[1]))
		})

		Context("when a commitish is not supplied", func() {
			It("updates the existing release", func() {
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
				Ω(githubClient.UpdateReleaseCallCount()).Should(Equal(1))

				updatedRelease := githubClient.UpdateReleaseArgsForCall(0)
				Ω(*updatedRelease.Name).Should(Equal("v0.3.12"))
				Ω(*updatedRelease.Body).Should(Equal("this is a great release"))
				Ω(updatedRelease.TargetCommitish).Should(Equal(github.String("1z22f1")))
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
				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(release.TargetCommitish).Should(Equal(github.String("a2f4a3")))
			})
		})

		Context("without a commitish", func() {
			It("creates a release on GitHub without the commitish", func() {
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
				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal("this is a great release"))
			})
		})

		Context("without a body", func() {
			It("works", func() {
				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
			})
		})

		It("always defaults to non-draft mode", func() {
			Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
			release := githubClient.CreateReleaseArgsForCall(0)

			Ω(*release.Draft).Should(Equal(false))
		})

		Context("when set as a draft release", func() {
			BeforeEach(func() {
				bodyPath := filepath.Join(sourcesDir, "body")
				file(bodyPath, "this is a great release")
				request.Params.Draft = true
			})

			It("creates a release on GitHub in draft mode", func() {
				Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
				release := githubClient.CreateReleaseArgsForCall(0)

				Ω(*release.Name).Should(Equal("v0.3.12"))
				Ω(*release.TagName).Should(Equal("0.3.12"))
				Ω(*release.Body).Should(Equal(""))
				Ω(*release.Draft).Should(Equal(true))
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
				Ω(githubClient.UploadReleaseAssetCallCount()).Should(Equal(1))
				release, name, file := githubClient.UploadReleaseAssetArgsForCall(0)

				Ω(*release.ID).Should(Equal(112))
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
		})
	})
})
