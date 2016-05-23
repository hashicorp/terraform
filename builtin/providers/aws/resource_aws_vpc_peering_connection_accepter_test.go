// make testacc TEST=./builtin/providers/aws/ TESTARGS='-run=TestAccAwsVPCPeeringConnectionAccepter_'
package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsVPCPeeringConnectionAccepter_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsVPCPeeringConnectionAccepterConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccAwsVPCPeeringConnectionAccepterCheckSomething(""),
				),
			},
		},
	})
}

func testAccAwsVPCPeeringConnectionAccepterCheckSomething(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		return nil
	}
}

const testAccAwsVPCPeeringConnectionAccepterConfig = `
`
