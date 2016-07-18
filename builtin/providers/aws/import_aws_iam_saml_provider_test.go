package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSIAMSamlProvider_importBasic(t *testing.T) {
	resourceName := "aws_iam_saml_provider.salesforce"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMSamlProviderDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMSamlProviderConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
