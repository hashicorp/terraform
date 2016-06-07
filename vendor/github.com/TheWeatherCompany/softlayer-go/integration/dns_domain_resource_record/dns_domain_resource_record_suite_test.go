package dns_resource_record

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServices(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration: DNS Domain resource record suite")
}
