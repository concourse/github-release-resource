package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/go-github/github"

	"github.com/concourse/github-release-resource"
	"github.com/concourse/github-release-resource/fakes"
)

var _ = Describe("Check Command", func() {
	var (
		githubClient *fakes.FakeGitHub
		command      *resource.CheckCommand

		returnedReleases []*github.RepositoryRelease
	)

	BeforeEach(func() {
		githubClient = &fakes.FakeGitHub{}
		command = resource.NewCheckCommand(githubClient)

		returnedReleases = []*github.RepositoryRelease{}
	})

	JustBeforeEach(func() {
		githubClient.ListReleasesReturns(returnedReleases, nil)
	})

	Context("when this is the first time that the resource has been run", func() {
		Context("when there are no releases", func() {
			BeforeEach(func() {
				returnedReleases = []*github.RepositoryRelease{}
			})

			It("returns no versions", func() {
				versions, err := command.Run(resource.CheckRequest{})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(BeEmpty())
			})
		})

		Context("when there are releases that get filtered out", func() {
			BeforeEach(func() {
				returnedReleases = []*github.RepositoryRelease{
					newDraftRepositoryRelease(1, "v0.1.4"),
				}
			})

			It("returns no versions", func() {
				versions, err := command.Run(resource.CheckRequest{})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(BeEmpty())
			})
		})

		Context("when there are releases", func() {
			BeforeEach(func() {
				returnedReleases = []*github.RepositoryRelease{
					newRepositoryRelease(1, "v0.4.0"),
					newRepositoryRelease(2, "0.1.3"),
					newRepositoryRelease(3, "v0.1.2"),
				}
			})

			It("outputs the most recent version only", func() {
				command := resource.NewCheckCommand(githubClient)

				response, err := command.Run(resource.CheckRequest{})
				Ω(err).ShouldNot(HaveOccurred())

				Ω(response).Should(HaveLen(1))
				Ω(response[0]).Should(Equal(resource.Version{
					Tag: "v0.4.0",
				}))
			})
		})
	})

	Context("when there are prior versions", func() {
		Context("when there are no releases", func() {
			BeforeEach(func() {
				returnedReleases = []*github.RepositoryRelease{}
			})

			It("returns no versions", func() {
				versions, err := command.Run(resource.CheckRequest{})
				Ω(err).ShouldNot(HaveOccurred())
				Ω(versions).Should(BeEmpty())
			})
		})

		Context("when there are releases", func() {
			Context("and there is a custom tag filter", func() {
				BeforeEach(func() {
					returnedReleases = []*github.RepositoryRelease{
						newRepositoryRelease(1, "package-0.1.4"),
						newRepositoryRelease(2, "package-0.4.0"),
						newRepositoryRelease(3, "package-0.1.3"),
						newRepositoryRelease(4, "package-0.1.2"),
					}
				})

				It("returns all of the versions that are newer", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "package-0.1.3",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{Tag: "package-0.1.3"},
						{Tag: "package-0.1.4"},
						{Tag: "package-0.4.0"},
					}))
				})
			})

			Context("and the releases do not contain a draft release", func() {
				BeforeEach(func() {
					returnedReleases = []*github.RepositoryRelease{
						newRepositoryRelease(1, "v0.1.4"),
						newRepositoryRelease(2, "0.4.0"),
						newRepositoryRelease(3, "v0.1.3"),
						newRepositoryRelease(4, "0.1.2"),
					}
				})

				It("returns an empty list if the lastet version has been checked", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "0.4.0",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(BeEmpty())
				})

				It("returns all of the versions that are newer", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "v0.1.3",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{Tag: "v0.1.3"},
						{Tag: "v0.1.4"},
						{Tag: "0.4.0"},
					}))
				})

				It("returns the latest version if the current version is not found", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "v3.4.5",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{Tag: "0.4.0"},
					}))
				})

				Context("when there are not-quite-semver versions", func() {
					BeforeEach(func() {
						returnedReleases = append(returnedReleases, newRepositoryRelease(5, "v1"))
						returnedReleases = append(returnedReleases, newRepositoryRelease(6, "v0"))
					})

					It("combines them with the semver versions in a reasonable order", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{
								Tag: "v0.1.3",
							},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "v0.1.3"},
							{Tag: "v0.1.4"},
							{Tag: "0.4.0"},
							{Tag: "v1"},
						}))
					})
				})
			})

			Context("and one of the releases is a draft", func() {
				BeforeEach(func() {
					returnedReleases = []*github.RepositoryRelease{
						newDraftRepositoryRelease(1, "v0.1.4"),
						newRepositoryRelease(2, "0.4.0"),
						newRepositoryRelease(3, "v0.1.3"),
					}
				})

				It("returns all of the versions that are newer, and not a draft", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "v0.1.3",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{Tag: "v0.1.3"},
						{Tag: "0.4.0"},
					}))
				})
			})

			Context("when pre releases are allowed and releases are not", func() {
				Context("and one of the releases is a final and another is a draft", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newDraftRepositoryRelease(1, "v0.1.4"),
							newRepositoryRelease(2, "0.4.0"),
							newPreReleaseRepositoryRelease(1, "v0.4.1-rc.10"),
							newPreReleaseRepositoryRelease(2, "0.4.1-rc.9"),
							newPreReleaseRepositoryRelease(3, "v0.4.1-rc.8"),
						}

					})

					It("returns all of the versions that are newer, and only pre relases", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "2"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: false},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "0.4.1-rc.9"},
							{Tag: "v0.4.1-rc.10"},
						}))
					})

					It("returns the latest prerelease version if the current version is not found", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "5"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: false},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "v0.4.1-rc.10"},
						}))
					})
				})

			})

			Context("when releases and pre releases are allowed", func() {
				Context("and final release is newer", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newDraftRepositoryRelease(1, "v0.1.4"),
							newRepositoryRelease(1, "0.3.9"),
							newRepositoryRelease(2, "0.4.0"),
							newRepositoryRelease(3, "v0.4.2"),
							newPreReleaseRepositoryRelease(1, "v0.4.1-rc.10"),
							newPreReleaseRepositoryRelease(2, "0.4.1-rc.9"),
							newPreReleaseRepositoryRelease(3, "v0.4.2-rc.1"),
						}

					})

					It("returns all of the versions that are newer, and are release and prerealse", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{Tag: "0.4.0"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "0.4.0"},
							{Tag: "0.4.1-rc.9"},
							{Tag: "v0.4.1-rc.10"},
							{Tag: "v0.4.2-rc.1"},
							{Tag: "v0.4.2"},
						}))
					})

					It("returns the latest release version if the current version is not found", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "5"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "v0.4.2"},
						}))
					})
				})

				Context("and prerelease is newer", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newDraftRepositoryRelease(1, "v0.1.4"),
							newRepositoryRelease(1, "0.3.9"),
							newRepositoryRelease(2, "0.4.0"),
							newRepositoryRelease(3, "v0.4.2"),
							newPreReleaseRepositoryRelease(1, "v0.4.1-rc.10"),
							newPreReleaseRepositoryRelease(2, "0.4.1-rc.9"),
							newPreReleaseRepositoryRelease(3, "v0.4.2-rc.1"),
							newPreReleaseRepositoryRelease(4, "v0.4.3-rc.1"),
						}

					})

					It("returns all of the versions that are newer, and are release and prerealse", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{Tag: "0.4.0"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "0.4.0"},
							{Tag: "0.4.1-rc.9"},
							{Tag: "v0.4.1-rc.10"},
							{Tag: "v0.4.2-rc.1"},
							{Tag: "v0.4.2"},
							{Tag: "v0.4.3-rc.1"},
						}))
					})

					It("returns the latest prerelease version if the current version is not found", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "5"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{Tag: "v0.4.3-rc.1"},
						}))
					})
				})

			})

			Context("when draft releases are allowed", func() {
				Context("and one of the releases is a final release", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newDraftRepositoryRelease(1, "v0.1.4"),
							newDraftRepositoryRelease(2, "v0.1.3"),
							newDraftRepositoryRelease(3, "v0.1.1"),
							newRepositoryRelease(2, "0.4.0"),
						}
					})

					It("returns all of the versions that are newer, and only draft", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "2"},
							Source:  resource.Source{Drafts: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{ID: "2"},
							{ID: "1"},
						}))
					})

					It("returns the latest draft version if the current version is not found", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "5"},
							Source:  resource.Source{Drafts: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{ID: "1"},
						}))
					})
				})

				Context("and non-of them are semver", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newDraftRepositoryRelease(1, "abc/d"),
							newDraftRepositoryRelease(2, "123*4"),
						}
					})

					It("returns all of the releases with semver resources", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{},
							Source:  resource.Source{Drafts: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{}))
					})
				})

				Context("and one of the releases is not a versioned draft release", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newDraftRepositoryRelease(1, "v0.1.4"),
							newDraftRepositoryRelease(2, ""),
							newDraftWithNilTagRepositoryRelease(3),
							newDraftRepositoryRelease(4, "asdf@example.com"),
						}
					})

					It("returns all of the releases with semver resources", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{},
							Source:  resource.Source{Drafts: true, PreRelease: false},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{ID: "1"},
						}))
					})
				})
			})
		})
	})
})
