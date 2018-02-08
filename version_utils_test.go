package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/concourse/github-release-resource"
)

var _ = Describe("VersionUtils", func() {
	Describe("FilterVersion", func() {
		It("should filter list of versions by tag", func() {
			inputVersions := []Version{{Tag: "0.1.1"}, {Tag: "0.2.1"}, {Tag: "0.3.0"}, {Tag: "0.3.1"}}
			versions, err := FilterVersions(inputVersions, "< 0.3.0")
			Ω(err).ToNot(HaveOccurred())
			Ω(versions).To(Equal([]Version{{Tag: "0.1.1"}, {Tag: "0.2.1"}}))
		})
		Context("When filter is blank", func() {
			It("should return the input versions", func() {
				inputVersions := []Version{{Tag: "0.1.1"}, {Tag: "0.2.1"}, {Tag: "0.3.0"}, {Tag: "0.3.1"}}
				versions, err := FilterVersions(inputVersions, "")
				Ω(err).ToNot(HaveOccurred())
				Ω(versions).To(Equal(inputVersions))
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
		It("should determine if the given versions bool value", func() {
			predicate := VersionPredicate{Condition: "<", Version: "4.0.0"}
			Ω(predicate.Apply(Version{Tag: "3.9.9"})).To(Equal(true))
		})
	})
})
