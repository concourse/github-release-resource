package resource_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/go-github/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/concourse/github-release-resource"
	"github.com/concourse/github-release-resource/fakes"
)

var _ = Describe("In Command", func() {
	var (
		command      *resource.InCommand
		githubClient *fakes.FakeGitHub
		server       *ghttp.Server

		inRequest resource.InRequest

		inResponse resource.InResponse
		inErr      error

		destDir string
	)

	BeforeEach(func() {
		var err error

		githubClient = &fakes.FakeGitHub{}
		command = resource.NewInCommand(githubClient, ioutil.Discard)

		destDir, err = ioutil.TempDir("", "github-release")
		Ω(err).ShouldNot(HaveOccurred())

		server = ghttp.NewServer()
		server.RouteToHandler("GET", "/example.txt", ghttp.RespondWith(200, "example.txt"))
		server.RouteToHandler("GET", "/example.rtf", ghttp.RespondWith(200, "example.rtf"))
		server.RouteToHandler("GET", "/example.wtf", ghttp.RespondWith(200, "example.wtf"))

		inRequest = resource.InRequest{}
	})

	JustBeforeEach(func() {
		inResponse, inErr = command.Run(destDir, inRequest)
	})

	AfterEach(func() {
		server.Close()
		Ω(os.RemoveAll(destDir)).Should(Succeed())
	})

	buildRelease := func(id int, tag string) github.RepositoryRelease {
		return github.RepositoryRelease{
			ID:      github.Int(id),
			TagName: github.String(tag),
			HTMLURL: github.String("http://google.com"),
			Name:    github.String("release-name"),
			Body:    github.String("*markdown*"),
		}
	}

	buildAsset := func(name string) github.ReleaseAsset {
		return github.ReleaseAsset{
			Name:               &name,
			BrowserDownloadURL: github.String(server.URL() + "/" + name),
		}
	}

	Context("when there are releases", func() {
		BeforeEach(func() {
			githubClient.ListReleasesReturns([]github.RepositoryRelease{
				buildRelease(2, "v0.35.0"),
				buildRelease(1, "v0.34.0"),
			}, nil)

			githubClient.ListReleaseAssetsReturns([]github.ReleaseAsset{
				buildAsset("example.txt"),
				buildAsset("example.rtf"),
				buildAsset("example.wtf"),
			}, nil)
		})

		Context("when a present version is specified", func() {
			BeforeEach(func() {
				inRequest.Version = &resource.Version{
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
					_, err := os.Stat(filepath.Join(destDir, "example.txt"))
					Ω(err).ShouldNot(HaveOccurred())

					_, err = os.Stat(filepath.Join(destDir, "example.rtf"))
					Ω(err).ShouldNot(HaveOccurred())

					_, err = os.Stat(filepath.Join(destDir, "example.wtf"))
					Ω(err).Should(HaveOccurred())
				})
			})

			Context("when an invalid asset filename glob is given", func() {
				BeforeEach(func() {
					inRequest.Params = resource.InParams{
						Globs: []string{`[`},
					}
				})

				It("returns an error", func() {
					Ω(inErr).Should(HaveOccurred())
				})
			})

			Context("when no globs are specified", func() {
				BeforeEach(func() {
					inRequest.Source = resource.Source{}
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
					_, err := os.Stat(filepath.Join(destDir, "example.txt"))
					Ω(err).ShouldNot(HaveOccurred())

					_, err = os.Stat(filepath.Join(destDir, "example.rtf"))
					Ω(err).ShouldNot(HaveOccurred())

					_, err = os.Stat(filepath.Join(destDir, "example.wtf"))
					Ω(err).ShouldNot(HaveOccurred())
				})
			})
		})

		Context("when the specified version is not available", func() {
			BeforeEach(func() {
				inRequest.Version = &resource.Version{
					Tag: "v0.36.0",
				}
			})

			It("returns an error", func() {
				Ω(inErr).Should(HaveOccurred())
			})
		})

		Context("when the version is not specified", func() {
			BeforeEach(func() {
				inRequest.Version = nil
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

			It("fetches from the latest release", func() {
				_, err := os.Stat(filepath.Join(destDir, "example.txt"))
				Ω(err).ShouldNot(HaveOccurred())

				_, err = os.Stat(filepath.Join(destDir, "example.rtf"))
				Ω(err).ShouldNot(HaveOccurred())

				_, err = os.Stat(filepath.Join(destDir, "example.wtf"))
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Context("when no releases are present", func() {
		BeforeEach(func() {
			githubClient.ListReleasesReturns([]github.RepositoryRelease{}, nil)
		})

		It("returns an error", func() {
			Ω(inErr).Should(HaveOccurred())
		})
	})
})
