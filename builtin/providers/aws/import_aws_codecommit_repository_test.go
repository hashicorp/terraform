package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCodeCommitRepository_importBasic(t *testing.T) {
	resName := "aws_codecommit_repository.test"
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCodeCommitRepositoryDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCodeCommitRepository_basic(rInt),
			},
			{
				ResourceName:      resName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
