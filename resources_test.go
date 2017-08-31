package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/concourse/github-release-resource"
)

var _ = Describe("Resources", func() {
	Describe("NewCheckRequest", func() {
		It("creates a new check request", func() {
			Expect(resource.NewCheckRequest()).To(Equal(resource.CheckRequest{
				Source: resource.Source{
					Release: true,
				},
			}))
		})
	})

	Describe("NewOutRequest", func() {
		It("creates a new check request", func() {
			Expect(resource.NewOutRequest()).To(Equal(resource.OutRequest{
				Source: resource.Source{
					Release: true,
				},
			}))
		})
	})

	Describe("NewInRequest", func() {
		It("creates a new check request", func() {
			Expect(resource.NewInRequest()).To(Equal(resource.InRequest{
				Source: resource.Source{
					Release: true,
				},
			}))
		})
	})
})
