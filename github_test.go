package resource_test

import (
	"fmt"
	"io/ioutil"
	"net/http"

	. "github.com/concourse/github-release-resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/go-github/github"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("GitHub Client", func() {
	var server *ghttp.Server
	var client *GitHubClient
	var source Source

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	JustBeforeEach(func() {
		source.GitHubAPIURL = server.URL()

		var err error
		client, err = NewGitHubClient(source)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		server.Close()
	})

	Context("with bad URLs", func() {
		BeforeEach(func() {
			source.AccessToken = "hello?"
		})

		It("returns an error if the API URL is bad", func() {
			source.GitHubAPIURL = ":"

			_, err := NewGitHubClient(source)
			Ω(err).Should(HaveOccurred())
		})

		It("returns an error if the API URL is bad", func() {
			source.GitHubUploadsURL = ":"

			_, err := NewGitHubClient(source)
			Ω(err).Should(HaveOccurred())
		})
	})

	Context("with an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				Owner:       "concourse",
				Repository:  "concourse",
				AccessToken: "abc123",
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases"),
					ghttp.RespondWith(200, "[]"),
					ghttp.VerifyHeaderKV("Authorization", "Bearer abc123"),
				),
			)
		})

		It("sends one", func() {
			_, err := client.ListReleases()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Context("without an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				Owner:      "concourse",
				Repository: "concourse",
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases"),
					ghttp.RespondWith(200, "[]"),
					ghttp.VerifyHeader(http.Header{"Authorization": nil}),
				),
			)
		})

		It("sends one", func() {
			_, err := client.ListReleases()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("when the source is configured with the deprecated user field", func() {
		BeforeEach(func() {
			source = Source{
				User:       "some-owner",
				Repository: "some-repo",
			}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/repos/some-owner/some-repo/releases"),
					ghttp.RespondWith(200, "[]"),
				),
			)
		})

		It("uses the provided user as the owner", func() {
			_, err := client.ListReleases()
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("ListReleases", func() {
		BeforeEach(func() {
			source = Source{
				Owner:      "concourse",
				Repository: "concourse",
			}
		})
		Context("When list of releases return more then 100 items", func() {
			BeforeEach(func() {
				var result []*github.RepositoryRelease
				for i := 1; i < 102; i++ {
					result = append(result, &github.RepositoryRelease{ID: github.Int(i)})

				}

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases", "per_page=100"),
						ghttp.RespondWithJSONEncoded(200, result[:100], http.Header{"Link": []string{`</releases?page=2>; rel="next"`}}),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases", "per_page=100&page=2"),
						ghttp.RespondWithJSONEncoded(200, result[100:]),
					),
				)
			})

			It("list releases", func() {
				releases, err := client.ListReleases()
				Ω(err).ShouldNot(HaveOccurred())
				Expect(releases).To(HaveLen(101))
				Expect(server.ReceivedRequests()).To(HaveLen(2))
			})
		})
	})

	Describe("GetRelease", func() {
		BeforeEach(func() {
			source = Source{
				Owner:      "concourse",
				Repository: "concourse",
			}
		})
		Context("When GitHub's rate limit has been exceeded", func() {
			BeforeEach(func() {
				rateLimitResponse := `{
          "message": "API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)",
          "documentation_url": "https://developer.github.com/v3/#rate-limiting"
        }`

				rateLimitHeaders := http.Header(map[string][]string{
					"X-RateLimit-Limit":     {"60"},
					"X-RateLimit-Remaining": {"0"},
					"X-RateLimit-Reset":     {"1377013266"},
				})

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases/20"),
						ghttp.RespondWith(403, rateLimitResponse, rateLimitHeaders),
					),
				)
			})

			It("Returns an appropriate error", func() {
				_, err := client.GetRelease(20)
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)"))
			})
		})
	})

	Describe("GetReleaseByTag", func() {
		BeforeEach(func() {
			source = Source{
				Owner:      "concourse",
				Repository: "concourse",
			}
		})

		Context("When GitHub's rate limit has been exceeded", func() {
			BeforeEach(func() {
				rateLimitResponse := `{
          "message": "API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)",
          "documentation_url": "https://developer.github.com/v3/#rate-limiting"
        }`

				rateLimitHeaders := http.Header(map[string][]string{
					"X-RateLimit-Limit":     {"60"},
					"X-RateLimit-Remaining": {"0"},
					"X-RateLimit-Reset":     {"1377013266"},
				})

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases/tags/some-tag"),
						ghttp.RespondWith(403, rateLimitResponse, rateLimitHeaders),
					),
				)
			})

			It("Returns an appropriate error", func() {
				_, err := client.GetReleaseByTag("some-tag")
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)"))
			})
		})

		Context("When GitHub responds successfully", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases/tags/some-tag"),
						ghttp.RespondWith(200, `{ "id": 1 }`),
					),
				)
			})

			It("Returns a populated github.RepositoryRelease", func() {
				expectedRelease := &github.RepositoryRelease{
					ID: github.Int(1),
				}

				release, err := client.GetReleaseByTag("some-tag")

				Ω(err).ShouldNot(HaveOccurred())
				Expect(release).To(Equal(expectedRelease))
			})
		})
	})

	Describe("ResolveTagToCommitSHA", func() {
		BeforeEach(func() {
			source = Source{
				Owner:      "concourse",
				Repository: "concourse",
			}
		})

		Context("When GitHub's rate limit has been exceeded", func() {
			BeforeEach(func() {
				rateLimitResponse := `{
          "message": "API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)",
          "documentation_url": "https://developer.github.com/v3/#rate-limiting"
        }`

				rateLimitHeaders := http.Header(map[string][]string{
					"X-RateLimit-Limit":     {"60"},
					"X-RateLimit-Remaining": {"0"},
					"X-RateLimit-Reset":     {"1377013266"},
				})

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/refs/tags/some-tag"),
						ghttp.RespondWith(403, rateLimitResponse, rateLimitHeaders),
					),
				)
			})

			It("Returns an appropriate error", func() {
				_, err := client.ResolveTagToCommitSHA("some-tag")
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)"))
			})
		})

		Context("When GitHub returns a lightweight tag", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/refs/tags/some-tag"),
						ghttp.RespondWith(200, `{ "ref": "refs/tags/some-tag", "object" : { "type": "commit", "sha": "some-sha"} }`),
					),
				)
			})

			It("Returns the associated commit SHA", func() {
				reference, err := client.ResolveTagToCommitSHA("some-tag")

				Ω(err).ShouldNot(HaveOccurred())
				Expect(reference).To(Equal("some-sha"))
			})
		})

		Context("When GitHub returns a reference to an annotated tag", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/refs/tags/some-tag"),
						ghttp.RespondWith(200, `{ "ref": "refs/tags/some-tag", "object" : { "type": "tag", "sha": "some-tag-sha"} }`),
					),
				)
			})

			Context("When GitHub returns the annotated tag", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/tags/some-tag-sha"),
							ghttp.RespondWith(200, `{ "object" : { "type": "commit", "sha": "some-sha"} }`),
						),
					)
				})

				It("Returns the associated commit SHA", func() {
					reference, err := client.ResolveTagToCommitSHA("some-tag")

					Ω(err).ShouldNot(HaveOccurred())
					Expect(reference).To(Equal("some-sha"))
				})
			})

			Context("When GitHub fails to fetch the annotated tag", func() {
				BeforeEach(func() {
					server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/tags/some-tag-sha"),
							ghttp.RespondWith(404, nil),
						),
					)
				})

				It("Returns an error", func() {
					_, err := client.ResolveTagToCommitSHA("some-tag")
					Ω(err).Should(HaveOccurred())
				})
			})

		})
	})

	Describe("DownloadReleaseAsset", func() {
		const (
			owner = "bob"
			repo  = "burgers"
		)

		var (
			assetID   int
			asset     github.ReleaseAsset
			assetPath string
		)

		BeforeEach(func() {
			source.Owner = owner
			source.Repository = repo
			source.AccessToken = "abc123"
			assetID = 42
			asset = github.ReleaseAsset{ID: &assetID}
			assetPath = fmt.Sprintf("/repos/%s/%s/releases/assets/%d", owner, repo, assetID)
		})

		var appendGetHandler = func(server *ghttp.Server, path string, statusCode int, body string, headers ...http.Header) {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", path),
					ghttp.RespondWith(statusCode, body, headers...),
					ghttp.VerifyHeaderKV("Authorization", "Bearer abc123"),
				),
			)
		}

		Context("when the asset can be downloaded directly", func() {
			Context("when the asset is downloaded successfully", func() {
				const (
					fileContents = "some-random-contents-from-github"
				)

				BeforeEach(func() {
					appendGetHandler(server, assetPath, 200, fileContents)
				})

				It("returns the correct body", func() {
					readCloser, err := client.DownloadReleaseAsset(asset)
					Expect(err).NotTo(HaveOccurred())
					defer readCloser.Close()

					body, err := ioutil.ReadAll(readCloser)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(body)).To(Equal(fileContents))
				})
			})

			Context("when there is an error downloading the asset", func() {
				BeforeEach(func() {
					appendGetHandler(server, assetPath, 401, "authorized personnel only")
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when the asset is behind a redirect", func() {
			const (
				redirectPath = "/the/redirect/path"
			)

			var locationHeader = func(url string) http.Header {
				header := make(http.Header)
				header.Add("Location", url)
				return header
			}

			BeforeEach(func() {
				appendGetHandler(server, assetPath, 307, "", locationHeader(redirectPath))
			})

			Context("when the redirect succeeds", func() {
				const (
					redirectFileContents = "some-random-contents-from-redirect"
				)

				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 200, redirectFileContents)
				})

				It("returns the body from the redirect request", func() {
					readCloser, err := client.DownloadReleaseAsset(asset)
					Expect(err).NotTo(HaveOccurred())
					defer readCloser.Close()

					body, err := ioutil.ReadAll(readCloser)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(body)).To(Equal(redirectFileContents))
				})

			})

			Context("when there is another redirect to a URL that succeeds", func() {
				const (
					redirectFileContents = "some-random-contents-from-redirect"
				)

				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 307, "", locationHeader("/somewhere-else"))
					appendGetHandler(server, "/somewhere-else", 200, redirectFileContents)
				})

				It("returns the body from the final redirect request", func() {
					readCloser, err := client.DownloadReleaseAsset(asset)
					Expect(err).NotTo(HaveOccurred())
					defer readCloser.Close()

					body, err := ioutil.ReadAll(readCloser)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(body)).To(Equal(redirectFileContents))
				})
			})

			Context("when the redirect request response is a 400", func() {
				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 400, "oops")
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 401", func() {
				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 401, "authorized personnel only")
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 403", func() {

				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 403, "authorized personnel only")
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 404", func() {
				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 404, "I don't know her")
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 500", func() {
				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 500, "boom")
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
