package resource_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	. "github.com/concourse/github-release-resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/go-github/v66/github"
	"github.com/onsi/gomega/ghttp"
)

const (
	multiPageRespEnterprise = `{
 "data": {
   "repository": {
     "releases": {
       "edges": [
         {
           "node": {
             "createdAt": "2010-10-01T00:58:07Z",
             "id": "MDc6UmVsZWFzZTMyMDk1MTAz",
             "name": "xyz",
             "publishedAt": "2010-10-02T15:39:53Z",
             "tagName": "xyz",
             "url": "https://github.com/xyz/xyz/releases/tag/xyz",
             "isDraft": false,
             "isPrerelease": false
           }
         },
         {
           "node": {
             "createdAt": "2010-08-27T13:55:36Z",
             "id": "MDc6UmVsZWFzZTMwMjMwNjU5",
             "name": "xyz",
             "publishedAt": "2010-08-27T17:18:06Z",
             "tagName": "xyz",
             "url": "https://github.com/xyz/xyz/releases/tag/xyz",
             "isDraft": false,
             "isPrerelease": false
           }
         }
       ],
       "pageInfo": {
         "endCursor": "Y3Vyc29yOnYyOpK5MjAyMC0xMC0wMVQwMjo1ODowNyswMjowMM4B6bt_",
         "hasNextPage": true
       }
     }
   }
 }
}`

	singlePageRespEnterprise = `{
  "data": {
    "repository": {
      "releases": {
        "edges": [
          {
            "node": {
              "createdAt": "2010-10-10T01:01:07Z",
              "id": "MDc6UmVsZWFzZTMzMjIyMjQz",
              "name": "xyq",
              "publishedAt": "2010-10-10T15:39:53Z",
              "tagName": "xyq",
              "url": "https://github.com/xyq/xyq/releases/tag/xyq",
              "isDraft": false,
              "isPrerelease": false
            }
          }
        ],
        "pageInfo": {
          "endCursor": "Y3Vyc29yOnYyOpK5MjAyMC0xMC0wMVQwMjo1ODowNyswMjowMM4B6bt_",
          "hasNextPage": false
        }
      }
    }
  }
}`
	invalidPageIdResp = `{
  "data": {
    "repository": {
      "releases": {
        "edges": [
          {
            "node": {
              "createdAt": "2010-10-10T01:01:07Z",
              "id": "MDc6UmVsZWFzZTMzMjZyzzzz",
              "databaseId":"3322224a",
              "name": "xyq",
              "publishedAt": "2010-10-10T15:39:53Z",
              "tagName": "xyq",
              "url": "https://github.com/xyq/xyq/releases/tag/xyq",
              "isDraft": false,
              "isPrerelease": false
            }
          }
        ],
        "pageInfo": {
          "endCursor": "Y3Vyc29yOnYyOpK5MjAyMC0xMC0wMVQwMjo1ODowNyswMjowMM4B6bt_",
          "hasNextPage": false
        }
      }
    }
  }
}`
	rateLimitMessage = `{
          "message": "API rate limit exceeded for 127.0.0.1. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)",
          "documentation_url": "https://developer.github.com/v3/#rate-limiting"
        }`
)

var _ = Describe("GitHub Client", func() {
	var server *ghttp.Server
	var client *GitHubClient
	var source Source

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	JustBeforeEach(func() {
		source.GitHubAPIURL = server.URL() + "/"

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

	Context("with good URLs", func() {
		var err error
		BeforeEach(func() {
			source = Source{
				Owner:      "concourse",
				Repository: "concourse",
			}
		})
		Context("given only the v3 API endpoint", func() {
			It("should replace v3 with graphql", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/api/graphql"),
						ghttp.RespondWith(200, singlePageRespEnterprise),
					),
				)

				source.GitHubAPIURL = server.URL() + "/api/v3"
				//setting the access token is how we ensure the v4 client is used
				source.AccessToken = "abc123"
				client, err = NewGitHubClient(source)
				Ω(err).ShouldNot(HaveOccurred())

				_, err := client.ListReleases()
				Ω(err).ShouldNot(HaveOccurred())
			})
			It("should always append graphql", func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/api/graphql"),
						ghttp.RespondWith(200, singlePageRespEnterprise),
					),
				)

				source.GitHubAPIURL = server.URL() + "/api/"
				//setting the access token is how we ensure the v4 client is used
				source.AccessToken = "abc123"
				client, err = NewGitHubClient(source)
				Ω(err).ShouldNot(HaveOccurred())

				_, err := client.ListReleases()
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Context("with an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				Owner:       "concourse",
				Repository:  "concourse",
				AccessToken: "abc123",
			}

			server.SetAllowUnhandledRequests(true)
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/graphql"),
					ghttp.RespondWith(200, singlePageRespEnterprise),
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

	Describe("ListReleases with access token", func() {
		BeforeEach(func() {
			source = Source{
				Owner:       "concourse",
				Repository:  "concourse",
				AccessToken: "test",
			}
		})
		Context("List graphql releases", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/graphql"),
						ghttp.RespondWith(200, multiPageRespEnterprise),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/graphql"),
						ghttp.RespondWith(200, singlePageRespEnterprise),
					),
				)
			})

			It("list releases", func() {
				releases, err := client.ListReleases()
				Ω(err).ShouldNot(HaveOccurred())
				Expect(releases).To(HaveLen(3))
				Expect(server.ReceivedRequests()).To(HaveLen(2))
				Expect(releases).To(Equal([]*github.RepositoryRelease{
					{TagName: github.String("xyz"), Name: github.String("xyz"), Draft: github.Bool(false), Prerelease: github.Bool(false), ID: github.Int64(32095103), CreatedAt: &github.Timestamp{time.Date(2010, time.October, 01, 00, 58, 07, 0, time.UTC)}, PublishedAt: &github.Timestamp{time.Date(2010, time.October, 02, 15, 39, 53, 0, time.UTC)}, URL: github.String("https://github.com/xyz/xyz/releases/tag/xyz")},
					{TagName: github.String("xyz"), Name: github.String("xyz"), Draft: github.Bool(false), Prerelease: github.Bool(false), ID: github.Int64(30230659), CreatedAt: &github.Timestamp{time.Date(2010, time.August, 27, 13, 55, 36, 0, time.UTC)}, PublishedAt: &github.Timestamp{time.Date(2010, time.August, 27, 17, 18, 06, 0, time.UTC)}, URL: github.String("https://github.com/xyz/xyz/releases/tag/xyz")},
					{TagName: github.String("xyq"), Name: github.String("xyq"), Draft: github.Bool(false), Prerelease: github.Bool(false), ID: github.Int64(33222243), CreatedAt: &github.Timestamp{time.Date(2010, time.October, 10, 01, 01, 07, 0, time.UTC)}, PublishedAt: &github.Timestamp{time.Date(2010, time.October, 10, 15, 39, 53, 0, time.UTC)}, URL: github.String("https://github.com/xyq/xyq/releases/tag/xyq")},
				}))
			})
		})

		Context("List graphql releases with bad id", func() {
			BeforeEach(func() {
				server.SetAllowUnhandledRequests(true)
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("POST", "/graphql"),
						ghttp.RespondWith(200, invalidPageIdResp),
					))
			})
			It("list releases with incorrect id", func() {
				_, err := client.ListReleases()
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Describe("ListReleases without access token", func() {
		BeforeEach(func() {
			source = Source{
				Owner:      "concourse",
				Repository: "concourse",
			}
		})
		Context("When list of releases return more then 100 items", func() {
			Context("List graphql releases", func() {
				BeforeEach(func() {
					var result []*github.RepositoryRelease
					for i := 1; i < 102; i++ {
						result = append(result, &github.RepositoryRelease{ID: github.Int64(int64(i))})
					}
					server.AppendHandlers(
						ghttp.CombineHandlers(ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases", "per_page=100"),
							ghttp.RespondWithJSONEncoded(200, result[:100], http.Header{"Link": []string{`</releases?page=2>; rel="next"`}}),
						),
						ghttp.CombineHandlers(ghttp.VerifyRequest("GET", "/repos/concourse/concourse/releases", "per_page=100&page=2"),
							ghttp.RespondWithJSONEncoded(200, result[100:])),
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
				rateLimitResponse := rateLimitMessage

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
				rateLimitResponse := rateLimitMessage

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
					ID: github.Int64(1),
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
				rateLimitResponse := rateLimitMessage

				rateLimitHeaders := http.Header(map[string][]string{
					"X-RateLimit-Limit":     {"60"},
					"X-RateLimit-Remaining": {"0"},
					"X-RateLimit-Reset":     {"1377013266"},
				})

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/ref/tags/some-tag"),
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
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/ref/tags/some-tag"),
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
						ghttp.VerifyRequest("GET", "/repos/concourse/concourse/git/ref/tags/some-tag"),
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
			assetID   int64
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

		var appendGetHandler = func(server *ghttp.Server, path string, statusCode int, body string, usesAuth bool, headers ...http.Header) {
			var authHeaderValue []string
			if usesAuth {
				authHeaderValue = []string{"Bearer abc123"}
			}
			server.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", path),
				ghttp.RespondWith(statusCode, body, headers...),
				ghttp.VerifyHeaderKV("Accept", "application/octet-stream"),
				ghttp.VerifyHeaderKV("Authorization", authHeaderValue...),
			))
		}

		var locationHeader = func(url string) http.Header {
			header := make(http.Header)
			header.Add("Location", url)
			return header
		}

		Context("when the asset can be downloaded directly", func() {
			Context("when the asset is downloaded successfully", func() {
				const (
					fileContents = "some-random-contents-from-github"
				)

				BeforeEach(func() {
					appendGetHandler(server, assetPath, 200, fileContents, true)
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
					appendGetHandler(server, assetPath, 401, "authorized personnel only", true)
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when the asset is behind a redirect", func() {
			const redirectPath = "/the/redirect/path"

			BeforeEach(func() {
				appendGetHandler(server, assetPath, 307, "", true, locationHeader(redirectPath))
			})

			Context("when the redirect succeeds", func() {
				const (
					redirectFileContents = "some-random-contents-from-redirect"
				)

				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 200, redirectFileContents, true)
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
					appendGetHandler(server, redirectPath, 307, "", true, locationHeader("/somewhere-else"))
					appendGetHandler(server, "/somewhere-else", 200, redirectFileContents, true)
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

			Context("when there is another redirect to an external server", func() {
				const (
					redirectFileContents = "some-random-contents-from-redirect"
				)

				var externalServer *ghttp.Server

				BeforeEach(func() {
					externalServer = ghttp.NewServer()
					u, err := url.Parse(externalServer.URL())
					Expect(err).NotTo(HaveOccurred())
					externalUrl := fmt.Sprintf("http://localhost:%s", u.Port())

					appendGetHandler(server, redirectPath, 307, "", true, locationHeader(externalUrl+"/somewhere-else"))
					appendGetHandler(externalServer, "/somewhere-else", 200, redirectFileContents, false)
				})

				It("downloads the file without the Authorization header", func() {
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
					appendGetHandler(server, redirectPath, 400, "oops", true)
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 401", func() {
				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 401, "authorized personnel only", true)
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 403", func() {

				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 403, "authorized personnel only", true)
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 404", func() {
				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 404, "I don't know her", true)
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("when the redirect request response is a 500", func() {
				BeforeEach(func() {
					appendGetHandler(server, redirectPath, 500, "boom", true)
				})

				It("returns an error", func() {
					_, err := client.DownloadReleaseAsset(asset)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when the asset is behind a redirect on an external server", func() {
			const (
				redirectFileContents = "some-random-contents-from-redirect"
			)

			var externalServer *ghttp.Server

			BeforeEach(func() {
				externalServer = ghttp.NewServer()

				appendGetHandler(server, assetPath, 307, "", true, locationHeader(externalServer.URL()+"/somewhere-else"))
				appendGetHandler(externalServer, "/somewhere-else", 200, redirectFileContents, false)
			})

			It("downloads the file without the Authorization header", func() {
				readCloser, err := client.DownloadReleaseAsset(asset)
				Expect(err).NotTo(HaveOccurred())
				defer readCloser.Close()

				body, err := ioutil.ReadAll(readCloser)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(body)).To(Equal(redirectFileContents))
			})
		})
	})
})
