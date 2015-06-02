package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCloudFrontWebDistribution(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontWebDistributionDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFrontWebDistributionConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontWebDistributionInitial(
						"aws_cloudfront.main",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudFrontWebDistributionUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontWebDistributionSecondary(
						"aws_cloudfront.main",
					),
				),
			},
		},
	})
}

func testAccCheckCloudFrontWebDistributionDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckCloudFrontWebDistributionInitial(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cloudFront, ok := s.RootModule().Resources[cloudFrontResource]
		if !ok {
			return fmt.Errorf("Not found: %s", cloudFrontResource)
		}

		if cloudFront.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		cloudfrontconn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

		req := &cloudfront.GetDistributionInput{
			ID: aws.String(cloudFront.Primary.ID),
		}

		_, err := cloudfrontconn.GetDistribution(req)
		if err != nil {
			return fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
		}

		return nil
	}
}

func testAccCheckCloudFrontWebDistributionSecondary(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cloudFront, ok := s.RootModule().Resources[cloudFrontResource]
		if !ok {
			return fmt.Errorf("Not found: %s", cloudFrontResource)
		}

		cloudfrontconn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

		req := &cloudfront.GetDistributionInput{
			ID: aws.String(cloudFront.Primary.ID),
		}

		res, err := cloudfrontconn.GetDistribution(req)
		if err != nil {
			return fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
		}

		if len(res.Distribution.DistributionConfig.Aliases.Items) != 1 {
			return fmt.Errorf("CloudFront failed updating aliases")
		}

		if *res.Distribution.DistributionConfig.Enabled != false {
			return fmt.Errorf("CloudFront failed updating enabled status")
		}

		return nil
	}
}

const testAccAWSCloudFrontWebDistributionConfig = `
resource "aws_cloudfront_web_distribution" "main" {
	default_origin = "main"

	origin {
    domain_name = "web.s3.amazonaws.com"
    id = "main"
  }
}
`

// CloudFront does not allow CNAME conflicts on the same account
var testAccAWSCloudFrontWebDistributionUpdate = fmt.Sprintf(`
resource "aws_cloudfront_web_distribution" "main" {
  enabled = false
	default_origin = "main"
  default_forward_cookie = "whitelist"
  default_whitelisted_cookies = ["session"]
  aliases = ["static-%d.example.com"]

  origin {
    domain_name = "web.s3.amazonaws.com"
    id = "main"
  }

  origin {
    domain_name = "images.s3.amazonaws.com"
    id = "images"
    origin_path = "/photos"
  }

  behavior {
    pattern = "images/*.jpg"
    origin = "images"
  }

  behavior {
    pattern = "images/*.png"
    origin = "images"
    minimum_ttl = 100
    whitelisted_cookies = ["test"]
  }
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
