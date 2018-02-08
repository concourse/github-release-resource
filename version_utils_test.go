package resource_test

import (
	"github.com/google/go-github/github"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/concourse/github-release-resource"
)

var _ = Describe("VersionUtils", func() {
	Describe("FilterByVersion", func() {
		var inputReleases []*github.RepositoryRelease

		BeforeEach(func() {
			inputReleases = []*github.RepositoryRelease{
				newRepositoryRelease(1, "0.1.1"),
				newRepositoryRelease(1, "0.2.1"),
				newRepositoryRelease(1, "0.3.0"),
				newRepositoryRelease(1, "0.3.1"),
			}
		})

		It("should filter releases by versions", func() {
			filteredReleases, err := FilterByVersion(inputReleases, "< 0.3.0")
			Ω(err).ToNot(HaveOccurred())

			expectedReleases := []*github.RepositoryRelease{
				newRepositoryRelease(1, "0.1.1"),
				newRepositoryRelease(1, "0.2.1"),
			}
			Ω(filteredReleases).To(Equal(expectedReleases))
		})

		Context("When filter is blank", func() {
			It("should return the input versions", func() {
				filteredReleases, err := FilterByVersion(inputReleases, "")
				Ω(err).ToNot(HaveOccurred())
				Ω(filteredReleases).To(Equal(inputReleases))
			})
		})
	})

	Describe("VersionFilter", func() {
		Describe("ParsePredicate", func() {
			It("should parse a '<' filter", func() {
				filter, err := ParsePredicate("< 3.1.1")
				Ω(err).ToNot(HaveOccurred())
				Ω(filter).To(Equal(VersionPredicate{Condition: "<", Version: "3.1.1"}))
			})
		})
	})

	Describe("Apply", func() {
		It("should return true for 3.9.9 < 4.0.0", func() {
			predicate := VersionPredicate{Condition: "<", Version: "4.0.0"}
			Ω(predicate.Apply("3.9.9")).To(Equal(true))
		})

		It("should return true for v3.9.9 < 4.0.0", func() {
			predicate := VersionPredicate{Condition: "<", Version: "4.0.0"}
			Ω(predicate.Apply("v3.9.9")).To(Equal(true))
		})
	})
})
