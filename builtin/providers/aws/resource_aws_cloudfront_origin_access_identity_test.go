package aws

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSCloudFrontOriginAccessIdentity_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontOriginAccessIdentityDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFrontOriginAccessIdentityConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontOriginAccessIdentityExistence("aws_cloudfront_origin_access_identity.origin_access_identity"),
					resource.TestCheckResourceAttr("aws_cloudfront_origin_access_identity.origin_access_identity", "comment", "some comment"),
					resource.TestMatchResourceAttr("aws_cloudfront_origin_access_identity.origin_access_identity",
						"caller_reference",
						regexp.MustCompile("^20[0-9]{2}.*")),
					resource.TestMatchResourceAttr("aws_cloudfront_origin_access_identity.origin_access_identity",
						"s3_canonical_user_id",
						regexp.MustCompile("^[a-z0-9]+")),
					resource.TestMatchResourceAttr("aws_cloudfront_origin_access_identity.origin_access_identity",
						"cloudfront_access_identity_path",
						regexp.MustCompile("^origin-access-identity/cloudfront/[A-Z0-9]+")),
				),
			},
		},
	})
}

func TestAccAWSCloudFrontOriginAccessIdentity_noComment(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontOriginAccessIdentityDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSCloudFrontOriginAccessIdentityNoCommentConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontOriginAccessIdentityExistence("aws_cloudfront_origin_access_identity.origin_access_identity"),
					resource.TestMatchResourceAttr("aws_cloudfront_origin_access_identity.origin_access_identity",
						"caller_reference",
						regexp.MustCompile("^20[0-9]{2}.*")),
					resource.TestMatchResourceAttr("aws_cloudfront_origin_access_identity.origin_access_identity",
						"s3_canonical_user_id",
						regexp.MustCompile("^[a-z0-9]+")),
					resource.TestMatchResourceAttr("aws_cloudfront_origin_access_identity.origin_access_identity",
						"cloudfront_access_identity_path",
						regexp.MustCompile("^origin-access-identity/cloudfront/[A-Z0-9]+")),
				),
			},
		},
	})
}

func testAccCheckCloudFrontOriginAccessIdentityDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudfront_origin_access_identity" {
			continue
		}

		params := &cloudfront.GetCloudFrontOriginAccessIdentityInput{
			Id: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetCloudFrontOriginAccessIdentity(params)
		if err == nil {
			return fmt.Errorf("CloudFront origin access identity was not deleted")
		}
	}

	return nil
}

func testAccCheckCloudFrontOriginAccessIdentityExistence(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("Not found: %s", r)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Id is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).cloudfrontconn

		params := &cloudfront.GetCloudFrontOriginAccessIdentityInput{
			Id: aws.String(rs.Primary.ID),
		}

		_, err := conn.GetCloudFrontOriginAccessIdentity(params)
		if err != nil {
			return fmt.Errorf("Error retrieving CloudFront distribution: %s", err)
		}
		return nil
	}
}

const testAccAWSCloudFrontOriginAccessIdentityConfig = `
resource "aws_cloudfront_origin_access_identity" "origin_access_identity" {
	comment = "some comment"
}
`

const testAccAWSCloudFrontOriginAccessIdentityNoCommentConfig = `
resource "aws_cloudfront_origin_access_identity" "origin_access_identity" {
}
`
