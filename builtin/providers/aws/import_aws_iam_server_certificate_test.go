package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSIAMServerCertificate_importBasic(t *testing.T) {
	resourceName := "aws_iam_server_certificate.test_cert"
	rInt := acctest.RandInt()
	resourceId := fmt.Sprintf("terraform-test-cert-%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckIAMServerCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccIAMServerCertConfig(rInt),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     resourceId,
				ImportStateVerifyIgnore: []string{
					"private_key"},
			},
		},
	})
}
