package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSNetworkAcl_importBasic(t *testing.T) {
	/*
		checkFn := func(s []*terraform.InstanceState) error {
			// Expect 2: acl, 2 rules
			if len(s) != 3 {
				return fmt.Errorf("bad states: %#v", s)
			}

			return nil
		}
	*/

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSNetworkAclDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSNetworkAclEgressNIngressConfig,
			},

			{
				ResourceName:      "aws_network_acl.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
