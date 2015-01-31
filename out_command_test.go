package resource_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/concourse/github-release-resource"
	"github.com/concourse/github-release-resource/fakes"

	"github.com/google/go-github/github"
)

var _ = Describe("Out Command", func() {
	var (
		command      *resource.OutCommand
		githubClient *fakes.FakeGitHub

		sourcesDir string
	)

	BeforeEach(func() {
		var err error

		githubClient = &fakes.FakeGitHub{}
		command = resource.NewOutCommand(githubClient)

		sourcesDir, err = ioutil.TempDir("", "github-release")
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		Ω(os.RemoveAll(sourcesDir)).Should(Succeed())
	})

	It("creates a release on GitHub", func() {
		namePath := filepath.Join(sourcesDir, "name")
		bodyPath := filepath.Join(sourcesDir, "body")
		tagPath := filepath.Join(sourcesDir, "tag")

		file(namePath, "v0.3.12")
		file(bodyPath, "this is a great release")
		file(tagPath, "0.3.12")

		request := resource.OutRequest{
			Params: resource.OutParams{
				NamePath: "name",
				BodyPath: "body",
				TagPath:  "tag",
			},
		}

		_, err := command.Run(sourcesDir, request)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
		release := githubClient.CreateReleaseArgsForCall(0)

		Ω(*release.Name).Should(Equal("v0.3.12"))
		Ω(*release.TagName).Should(Equal("0.3.12"))
		Ω(*release.Body).Should(Equal("this is a great release"))
	})

	It("works without a body", func() {
		namePath := filepath.Join(sourcesDir, "name")
		tagPath := filepath.Join(sourcesDir, "tag")

		file(namePath, "v0.3.12")
		file(tagPath, "0.3.12")

		request := resource.OutRequest{
			Params: resource.OutParams{
				NamePath: "name",
				TagPath:  "tag",
			},
		}

		_, err := command.Run(sourcesDir, request)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(githubClient.CreateReleaseCallCount()).Should(Equal(1))
		release := githubClient.CreateReleaseArgsForCall(0)

		Ω(*release.Name).Should(Equal("v0.3.12"))
		Ω(*release.TagName).Should(Equal("0.3.12"))
		Ω(*release.Body).Should(Equal(""))
	})

	It("uploads matching file globs", func() {
		namePath := filepath.Join(sourcesDir, "name")
		bodyPath := filepath.Join(sourcesDir, "body")
		tagPath := filepath.Join(sourcesDir, "tag")

		file(namePath, "v0.3.12")
		file(bodyPath, "this is a great release")
		file(tagPath, "0.3.12")

		globMatching := filepath.Join(sourcesDir, "great-file.tgz")
		globNotMatching := filepath.Join(sourcesDir, "bad-file.txt")

		file(globMatching, "matching")
		file(globNotMatching, "not matching")

		githubClient.CreateReleaseStub = func(gh *github.RepositoryRelease) (*github.RepositoryRelease, error) {
			gh.ID = github.Int(112)
			return gh, nil
		}

		request := resource.OutRequest{
			Params: resource.OutParams{
				NamePath: "name",
				BodyPath: "body",
				TagPath:  "tag",

				Globs: []string{
					"*.tgz",
				},
			},
		}

		_, err := command.Run(sourcesDir, request)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(githubClient.UploadReleaseAssetCallCount()).Should(Equal(1))
		release, name, file := githubClient.UploadReleaseAssetArgsForCall(0)

		Ω(*release.ID).Should(Equal(112))
		Ω(name).Should(Equal("great-file.tgz"))
		Ω(file.Name()).Should(Equal(filepath.Join(sourcesDir, "great-file.tgz")))
	})
})

func file(path, contents string) {
	Ω(ioutil.WriteFile(path, []byte(contents), 0644)).Should(Succeed())
}
