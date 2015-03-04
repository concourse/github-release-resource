package resource_test

import (
	"net/http"

	. "github.com/concourse/github-release-resource"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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

	Context("with an OAuth Token", func() {
		BeforeEach(func() {
			source = Source{
				User:        "concourse",
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
				User:       "concourse",
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
})
