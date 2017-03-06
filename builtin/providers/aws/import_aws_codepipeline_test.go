package aws

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSCodePipeline_Import_basic(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Environment variable GITHUB_TOKEN is not set")
	}

	name := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSCodePipelineDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCodePipelineConfig_basic(name),
			},

			resource.TestStep{
				ResourceName:      "aws_codepipeline.bar",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
