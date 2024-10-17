package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/google/go-github/v66/github"

	resource "github.com/concourse/github-release-resource"
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

			Context("and releases are ordered by version", func() {
				It("returns no versions", func() {
					versions, err := command.Run(resource.CheckRequest{})
					Ω(err).ShouldNot(HaveOccurred())
					Ω(versions).Should(BeEmpty())
				})
			})

			Context("and releases are ordered by time", func() {
				It("returns no versions", func() {
					versions, err := command.Run(resource.CheckRequest{
						Source: resource.Source{OrderBy: "time"},
					})
					Ω(err).ShouldNot(HaveOccurred())
					Ω(versions).Should(BeEmpty())
				})
			})
		})

		Context("when there are releases", func() {
			BeforeEach(func() {
				returnedReleases = []*github.RepositoryRelease{
					newRepositoryReleaseWithCreatedTime(1, "v0.4.0", 2),
					newRepositoryReleaseWithCreatedTime(2, "v0.1.3", 3),
					newRepositoryReleaseWithCreatedTime(3, "v0.1.2", 1),
				}
			})

			Context("and releases are ordered by version", func() {
				It("outputs the most recent version only", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(HaveLen(1))
					Ω(response[0]).Should(Equal(newVersionWithTimestamp(1, "v0.4.0", 2)))
				})
			})

			Context("and releases are ordered by time", func() {
				It("outputs the most recent time only", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Source: resource.Source{OrderBy: "time"},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(HaveLen(1))
					Ω(response[0]).Should(Equal(newVersionWithTimestamp(2, "v0.1.3", 3)))
				})
			})

			Context("when there is a semver constraint", func() {
				BeforeEach(func() {
					returnedReleases = []*github.RepositoryRelease{
						newRepositoryReleaseWithCreatedTime(1, "v0.4.0", 2),
						newRepositoryReleaseWithCreatedTime(2, "0.1.3", 3),
						newRepositoryReleaseWithCreatedTime(3, "v0.1.2", 1),
						newRepositoryReleaseWithCreatedTime(4, "invalid-semver", 4),
					}
				})

				It("keeps only those versions matching the constraint", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Source: resource.Source{SemverConstraint: "0.1.x"},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(HaveLen(1))
					Ω(response[0]).Should(Equal(newVersionWithTimestamp(2, "0.1.3", 3)))
				})

				Context("when there is a custom tag filter", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newRepositoryReleaseWithCreatedTime(1, "foo-0.4.0", 2),
							newRepositoryReleaseWithCreatedTime(2, "foo-0.1.3", 3),
							newRepositoryReleaseWithCreatedTime(3, "foo-0.1.2", 1),
							newRepositoryReleaseWithCreatedTime(4, "0.1.4", 4),
						}
					})

					It("uses the filter", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Source: resource.Source{
								SemverConstraint: "0.1.x",
								TagFilter:        "foo-(.*)",
							},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(HaveLen(1))
						Ω(response[0]).Should(Equal(newVersionWithTimestamp(2, "foo-0.1.3", 3)))
					})
				})
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
						{ID: "3", Tag: "package-0.1.3"},
						{ID: "1", Tag: "package-0.1.4"},
						{ID: "2", Tag: "package-0.4.0"},
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

				It("returns the current version if it is also the latest", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "0.4.0",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{ID: "2", Tag: "0.4.0"},
					}))
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
						{ID: "3", Tag: "v0.1.3"},
						{ID: "1", Tag: "v0.1.4"},
						{ID: "2", Tag: "0.4.0"},
					}))
				})

				It("returns all newer versions even when current version not found", func() {
					command := resource.NewCheckCommand(githubClient)

					response, err := command.Run(resource.CheckRequest{
						Version: resource.Version{
							Tag: "v0.1.4-beta",
						},
					})
					Ω(err).ShouldNot(HaveOccurred())

					Ω(response).Should(Equal([]resource.Version{
						{ID: "1", Tag: "v0.1.4"},
						{ID: "2", Tag: "0.4.0"},
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
						{ID: "2", Tag: "0.4.0"},
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
							{ID: "3", Tag: "v0.1.3"},
							{ID: "1", Tag: "v0.1.4"},
							{ID: "2", Tag: "0.4.0"},
							{ID: "5", Tag: "v1"},
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
						{ID: "3", Tag: "v0.1.3"},
						{ID: "2", Tag: "0.4.0"},
					}))
				})
			})

			Context("when pre releases are allowed and releases are not", func() {
				Context("and one of the releases is a final and another is a draft", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newDraftRepositoryRelease(1, "v0.1.4"),
							newRepositoryRelease(2, "0.4.0"),
							newPreReleaseRepositoryRelease(3, "v0.4.1-rc.10"),
							newPreReleaseRepositoryRelease(4, "0.4.1-rc.9"),
							newPreReleaseRepositoryRelease(5, "v0.4.1-rc.8"),
						}

					})

					It("returns all of the versions that are newer, and only pre relases", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "3", Tag: "0.4.1-rc.9"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: false},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{ID: "4", Tag: "0.4.1-rc.9"},
							{ID: "3", Tag: "v0.4.1-rc.10"},
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
							{ID: "3", Tag: "v0.4.1-rc.10"},
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
							newPreReleaseRepositoryRelease(4, "v0.4.1-rc.10"),
							newPreReleaseRepositoryRelease(5, "0.4.1-rc.9"),
							newPreReleaseRepositoryRelease(6, "v0.4.2-rc.1"),
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
							{ID: "2", Tag: "0.4.0"},
							{ID: "5", Tag: "0.4.1-rc.9"},
							{ID: "4", Tag: "v0.4.1-rc.10"},
							{ID: "6", Tag: "v0.4.2-rc.1"},
							{ID: "3", Tag: "v0.4.2"},
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
							{ID: "3", Tag: "v0.4.2"},
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
							newPreReleaseRepositoryRelease(4, "v0.4.1-rc.10"),
							newPreReleaseRepositoryRelease(5, "0.4.1-rc.9"),
							newPreReleaseRepositoryRelease(6, "v0.4.2-rc.1"),
							newPreReleaseRepositoryRelease(7, "v0.4.3-rc.1"),
						}

					})

					It("returns all of the versions that are newer, and are release and prerelease", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{Tag: "0.4.0"},
							Source:  resource.Source{Drafts: false, PreRelease: true, Release: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{ID: "2", Tag: "0.4.0"},
							{ID: "5", Tag: "0.4.1-rc.9"},
							{ID: "4", Tag: "v0.4.1-rc.10"},
							{ID: "6", Tag: "v0.4.2-rc.1"},
							{ID: "3", Tag: "v0.4.2"},
							{ID: "7", Tag: "v0.4.3-rc.1"},
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
							{ID: "7", Tag: "v0.4.3-rc.1"},
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
							Version: resource.Version{Tag: "v0.1.3"},
							Source:  resource.Source{Drafts: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{ID: "2", Tag: "v0.1.3"},
							{ID: "1", Tag: "v0.1.4"},
						}))
					})

					It("returns all newer draft versions even if current version is not found", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{Tag: "v0.1.2"},
							Source:  resource.Source{Drafts: true},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							{ID: "2", Tag: "v0.1.3"},
							{ID: "1", Tag: "v0.1.4"},
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
							{ID: "1", Tag: "v0.1.4"},
						}))
					})
				})
			})

			Context("ordered by time", func() {
				Context("with created time only", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newRepositoryReleaseWithCreatedTime(1, "v0.1.1", 1),
							newRepositoryReleaseWithCreatedTime(2, "v0.2.1", 2),
							newRepositoryReleaseWithCreatedTime(3, "v0.1.3", 3),
							newRepositoryReleaseWithCreatedTime(4, "v0.1.4", 4),
						}
					})
					It("returns releases with newer created time", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: newVersionWithTimestamp(3, "v0.1.3", 3),
							Source:  resource.Source{OrderBy: "time"},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							newVersionWithTimestamp(3, "v0.1.3", 3),
							newVersionWithTimestamp(4, "v0.1.4", 4),
						}))
					})
				})
				Context("with published time only", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newRepositoryReleaseWithPublishedTime(1, "v0.1.1", 1),
							newRepositoryReleaseWithPublishedTime(2, "v0.2.1", 2),
							newRepositoryReleaseWithPublishedTime(3, "v0.1.3", 3),
							newRepositoryReleaseWithPublishedTime(4, "v0.1.4", 4),
						}
					})
					It("returns releases with newer published time", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: newVersionWithTimestamp(3, "v0.1.3", 3),
							Source:  resource.Source{OrderBy: "time"},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							newVersionWithTimestamp(3, "v0.1.3", 3),
							newVersionWithTimestamp(4, "v0.1.4", 4),
						}))
					})
				})
				Context("with created and published time", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newRepositoryReleaseWithCreatedAndPublishedTime(1, "v0.1.1", 1, 5),
							newRepositoryReleaseWithCreatedAndPublishedTime(2, "v0.2.1", 2, 4),
							newRepositoryReleaseWithCreatedAndPublishedTime(3, "v0.1.3", 4, 2),
							newRepositoryReleaseWithCreatedAndPublishedTime(4, "v0.1.4", 5, 1),
						}
					})
					It("returns releases with newer published time", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: newVersionWithTimestamp(2, "v0.2.1", 4),
							Source:  resource.Source{OrderBy: "time"},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							newVersionWithTimestamp(2, "v0.2.1", 4),
							newVersionWithTimestamp(1, "v0.1.1", 5),
						}))
					})
					It("returns releases with newer published time even when current version not found", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: newVersionWithTimestamp(9, "v1.0.0", 3),
							Source:  resource.Source{OrderBy: "time"},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							newVersionWithTimestamp(2, "v0.2.1", 4),
							newVersionWithTimestamp(1, "v0.1.1", 5),
						}))
					})
					It("returns release with latest published time when request has no timestamp", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: resource.Version{ID: "2", Tag: "v0.2.1"},
							Source:  resource.Source{OrderBy: "time"},
						})
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal([]resource.Version{
							newVersionWithTimestamp(1, "v0.1.1", 5),
						}))
					})

				})
				Context("without time", func() {
					BeforeEach(func() {
						returnedReleases = []*github.RepositoryRelease{
							newRepositoryRelease(1, "v0.1.1"),
							newRepositoryRelease(2, "v0.2.1"),
							newRepositoryRelease(3, "v0.1.3"),
							newRepositoryRelease(4, "v0.1.4"),
						}
					})
					It("returns empty list", func() {
						command := resource.NewCheckCommand(githubClient)

						response, err := command.Run(resource.CheckRequest{
							Version: newVersionWithTimestamp(2, "v0.2.1", 3),
							Source:  resource.Source{OrderBy: "time"},
						})
						Ω(err).ShouldNot(HaveOccurred())
						Ω(response).Should(Equal([]resource.Version{}))
					})
				})
			})
		})
	})
})
