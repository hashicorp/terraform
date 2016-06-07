package security_ssh_key_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testhelpers "github.com/TheWeatherCompany/softlayer-go/test_helpers"
)

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration: Security SSH key suite")
}

func cleanUpTestResources() {
	err := testhelpers.FindAndDeleteTestSshKeys()
	Expect(err).ToNot(HaveOccurred())
}
