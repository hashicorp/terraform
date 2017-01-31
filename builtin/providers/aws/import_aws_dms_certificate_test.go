package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAwsDmsCertificateImport(t *testing.T) {
	resourceName := "aws_dms_certificate.dms_certificate"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsCertificateDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsCertificateConfig(acctest.RandString(8)),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
