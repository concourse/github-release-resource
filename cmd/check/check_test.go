package main_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Check", func() {
	It("outputs an empty JSON array", func() {
		command := exec.Command(checkPath)

		check, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())

		Eventually(check).Should(gbytes.Say(`\[\]`))
		Eventually(check).Should(gexec.Exit(0))
	})
})
