package aws

import (
	//	"fmt"
	"testing"

	//	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSIAMServerCertificate_importBasic(t *testing.T) {
	resourceName := "aws_iam_server_certificate.test_cert"

	resource.Test(t, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccIAMServerCertConfig,
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"private_key"},
			},
		},
	})
}
