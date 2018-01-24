package resource_test

import (
	"net/http"

	. "github.com/concourse/github-release-resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
	"github.com/google/go-github/github"
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
					ID:         github.Int(1),
				}

				release, err := client.GetReleaseByTag("some-tag")

				Ω(err).ShouldNot(HaveOccurred())
				Expect(release).To(Equal(expectedRelease))
			})
		})
	})

	Describe("GetRef", func() {
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
				_, err := client.GetRef("some-tag")
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(ContainSubstring("API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)"))
			})
		})

		Context("When GitHub responds successfully", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/refs/tags/some-tag"),
						ghttp.RespondWith(200, `{ "ref": "refs/tags/some-tag" }`),
					),
				)
			})

			It("Returns a populated github.Reference", func() {
				expectedReference := &github.Reference{
					Ref: github.String("refs/tags/some-tag"),
				}

				reference, err := client.GetRef("some-tag")

				Ω(err).ShouldNot(HaveOccurred())
				Expect(reference).To(Equal(expectedReference))
			})
		})
	})
})
