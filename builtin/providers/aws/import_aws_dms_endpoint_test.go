package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAwsDmsEndpointImport(t *testing.T) {
	resourceName := "aws_dms_endpoint.dms_endpoint"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsEndpointConfig(acctest.RandString(8)),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}
