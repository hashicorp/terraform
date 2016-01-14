package aws

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudFront_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFrontConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudfrontExistance(
						"aws_cloudfront_web_distribution.main",
					),
				),
			},
			resource.TestStep{
				Config: testAccAWSCloudFrontUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudfrontExistance(
						"aws_cloudfront_web_distribution.main",
					),
					testAccCheckCloudfrontCheckDistributionDisabled(
						"aws_cloudfront_web_distribution.main",
					),
					testAccCheckCloudfrontCheckDistributionAlias(
						"aws_cloudfront_web_distribution.main",
					),
				),
			},
		},
	})
}

func testAccCheckCloudFrontDestroy(s *terraform.State) error {
	if len(s.RootModule().Resources) > 0 {
		return fmt.Errorf("Expected all resources to be gone, but found: %#v", s.RootModule().Resources)
	}

	return nil
}

func testAccCheckCloudfrontExistance(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testAccAuxCloudfrontGetDistributionConfig(s, cloudFrontResource)

		return err
	}
}

func testAccCheckCloudfrontCheckDistributionDisabled(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dist, _ := testAccAuxCloudfrontGetDistributionConfig(s, cloudFrontResource)

		if *dist.DistributionConfig.Enabled != false {
			return fmt.Errorf("CloudFront distribution should be disabled")
		}

		return nil
	}
}

func testAccCheckCloudfrontCheckDistributionAlias(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		dist, _ := testAccAuxCloudfrontGetDistributionConfig(s, cloudFrontResource)

		if len(dist.DistributionConfig.Aliases.Items) != 1 {
			return fmt.Errorf("CloudFront failed updating aliases")
		}

		return nil
	}
}

func testAccAuxCloudfrontGetDistributionConfig(s *terraform.State, cloudFrontResource string) (*cloudfront.Distribution, error) {
	cf, ok := s.RootModule().Resources[cloudFrontResource]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", cloudFrontResource)
	}

	if cf.Primary.ID == "" {
		return nil, fmt.Errorf("No Id is set")
	}

	cloudfrontconn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

	req := &cloudfront.GetDistributionInput{
		Id: aws.String(cf.Primary.ID),
	}

	res, err := cloudfrontconn.GetDistribution(req)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
	}

	return res.Distribution, nil
}

const testAccAWSCloudFrontConfig = `
resource "aws_cloudfront_web_distribution" "main" {
  origin_domain_name = "fileserver.example.com"
}
`

// CloudFront does not allow CNAME conflicts on the same account
var testAccAWSCloudFrontUpdate = fmt.Sprintf(`
resource "aws_cloudfront_web_distribution" "main" {
	enabled = false
  origin_domain_name = "fileserver.example.com"
	aliases = ["static-%d.example.com"]
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
