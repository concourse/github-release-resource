package resource_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestGithubReleaseResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github Release Resource Suite")
}
