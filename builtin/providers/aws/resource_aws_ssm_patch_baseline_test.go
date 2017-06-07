package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSSMPatchBaseline_basic(t *testing.T) {
	name := acctest.RandString(10)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSSMPatchBaselineDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSSSMPatchBaselineBasicConfig(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMPatchBaselineExists("aws_ssm_patch_baseline.foo"),
					resource.TestCheckResourceAttr(
						"aws_ssm_patch_baseline.foo", "approved_patches.#", "1"),
					resource.TestCheckResourceAttr(
						"aws_ssm_patch_baseline.foo", "approved_patches.2062620480", "KB123456"),
					resource.TestCheckResourceAttr(
						"aws_ssm_patch_baseline.foo", "name", fmt.Sprintf("patch-baseline-%s", name)),
				),
			},
			{
				Config: testAccAWSSSMPatchBaselineBasicConfigUpdated(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSSMPatchBaselineExists("aws_ssm_patch_baseline.foo"),
					resource.TestCheckResourceAttr(
						"aws_ssm_patch_baseline.foo", "approved_patches.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_ssm_patch_baseline.foo", "approved_patches.2062620480", "KB123456"),
					resource.TestCheckResourceAttr(
						"aws_ssm_patch_baseline.foo", "approved_patches.2291496788", "KB456789"),
					resource.TestCheckResourceAttr(
						"aws_ssm_patch_baseline.foo", "name", fmt.Sprintf("updated-patch-baseline-%s", name)),
				),
			},
		},
	})
}

func testAccCheckAWSSSMPatchBaselineExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SSM Patch Baseline ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).ssmconn

		resp, err := conn.DescribePatchBaselines(&ssm.DescribePatchBaselinesInput{
			Filters: []*ssm.PatchOrchestratorFilter{
				{
					Key:    aws.String("NAME_PREFIX"),
					Values: []*string{aws.String(rs.Primary.Attributes["name"])},
				},
			},
		})

		for _, i := range resp.BaselineIdentities {
			if *i.BaselineId == rs.Primary.ID {
				return nil
			}
		}
		if err != nil {
			return err
		}

		return fmt.Errorf("No AWS SSM Patch Baseline found")
	}
}

func testAccCheckAWSSSMPatchBaselineDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ssmconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_ssm_patch_baseline" {
			continue
		}

		out, err := conn.DescribePatchBaselines(&ssm.DescribePatchBaselinesInput{
			Filters: []*ssm.PatchOrchestratorFilter{
				{
					Key:    aws.String("NAME_PREFIX"),
					Values: []*string{aws.String(rs.Primary.Attributes["name"])},
				},
			},
		})

		if err != nil {
			return err
		}

		if len(out.BaselineIdentities) > 0 {
			return fmt.Errorf("Expected AWS SSM Patch Baseline to be gone, but was still found")
		}

		return nil
	}

	return nil
}

func testAccAWSSSMPatchBaselineBasicConfig(rName string) string {
	return fmt.Sprintf(`

resource "aws_ssm_patch_baseline" "foo" {
  name  = "patch-baseline-%s"
  approved_patches = ["KB123456"]
}

`, rName)
}

func testAccAWSSSMPatchBaselineBasicConfigUpdated(rName string) string {
	return fmt.Sprintf(`

resource "aws_ssm_patch_baseline" "foo" {
  name  = "updated-patch-baseline-%s"
  approved_patches = ["KB123456","KB456789"]
}

`, rName)
}
