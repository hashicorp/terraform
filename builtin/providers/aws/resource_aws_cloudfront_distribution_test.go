package aws

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

// TestAccAWSCloudFrontDistribution_S3Origin runs an
// aws_cloudfront_distribution acceptance test with a single S3 origin.
//
// If you are testing manually and can't wait for deletion, set the
// TF_TEST_CLOUDFRONT_RETAIN environment variable.
func TestAccAWSCloudFrontDistribution_S3Origin(t *testing.T) {
	ri := acctest.RandInt()
	testConfig := fmt.Sprintf(testAccAWSCloudFrontDistributionS3Config, ri, originBucket, logBucket, testAccAWSCloudFrontDistributionRetainConfig())
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.s3_distribution",
					),
					resource.TestCheckResourceAttr(
						"aws_cloudfront_distribution.s3_distribution",
						"hosted_zone_id",
						"Z2FDTNDATAQYW2",
					),
				),
			},
		},
	})
}

func TestAccAWSCloudFrontDistribution_S3OriginWithTags(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAWSCloudFrontDistributionS3ConfigWithTags, ri, originBucket, logBucket, testAccAWSCloudFrontDistributionRetainConfig())
	postConfig := fmt.Sprintf(testAccAWSCloudFrontDistributionS3ConfigWithTagsUpdated, ri, originBucket, logBucket, testAccAWSCloudFrontDistributionRetainConfig())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.s3_distribution",
					),
					resource.TestCheckResourceAttr(
						"aws_cloudfront_distribution.s3_distribution", "tags.%", "2"),
					resource.TestCheckResourceAttr(
						"aws_cloudfront_distribution.s3_distribution", "tags.environment", "production"),
					resource.TestCheckResourceAttr(
						"aws_cloudfront_distribution.s3_distribution", "tags.account", "main"),
				),
			},
			{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.s3_distribution",
					),
					resource.TestCheckResourceAttr(
						"aws_cloudfront_distribution.s3_distribution", "tags.%", "1"),
					resource.TestCheckResourceAttr(
						"aws_cloudfront_distribution.s3_distribution", "tags.environment", "dev"),
				),
			},
		},
	})
}

// TestAccAWSCloudFrontDistribution_customOriginruns an
// aws_cloudfront_distribution acceptance test with a single custom origin.
//
// If you are testing manually and can't wait for deletion, set the
// TF_TEST_CLOUDFRONT_RETAIN environment variable.
func TestAccAWSCloudFrontDistribution_customOrigin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontDistributionCustomConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.custom_distribution",
					),
				),
			},
		},
	})
}

// TestAccAWSCloudFrontDistribution_multiOrigin runs an
// aws_cloudfront_distribution acceptance test with multiple origins.
//
// If you are testing manually and can't wait for deletion, set the
// TF_TEST_CLOUDFRONT_RETAIN environment variable.
func TestAccAWSCloudFrontDistribution_multiOrigin(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontDistributionMultiOriginConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.multi_origin_distribution",
					),
				),
			},
		},
	})
}

// TestAccAWSCloudFrontDistribution_noOptionalItemsConfig runs an
// aws_cloudfront_distribution acceptance test with no optional items set.
//
// If you are testing manually and can't wait for deletion, set the
// TF_TEST_CLOUDFRONT_RETAIN environment variable.
func TestAccAWSCloudFrontDistribution_noOptionalItemsConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontDistributionNoOptionalItemsConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.no_optional_items",
					),
				),
			},
		},
	})
}

// TestAccAWSCloudFrontDistribution_HTTP11Config runs an
// aws_cloudfront_distribution acceptance test with the HTTP version set to
// 1.1.
//
// If you are testing manually and can't wait for deletion, set the
// TF_TEST_CLOUDFRONT_RETAIN environment variable.
func TestAccAWSCloudFrontDistribution_HTTP11Config(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontDistributionHTTP11Config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.http_1_1",
					),
				),
			},
		},
	})
}

func TestAccAWSCloudFrontDistribution_IsIPV6EnabledConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontDistributionIsIPV6EnabledConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.is_ipv6_enabled",
					),
					resource.TestCheckResourceAttr(
						"aws_cloudfront_distribution.is_ipv6_enabled", "is_ipv6_enabled", "true"),
				),
			},
		},
	})
}

func TestAccAWSCloudFrontDistribution_noCustomErrorResponseConfig(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCloudFrontDistributionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSCloudFrontDistributionNoCustomErroResponseInfo,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCloudFrontDistributionExistence(
						"aws_cloudfront_distribution.no_custom_error_responses",
					),
				),
			},
		},
	})
}

func TestResourceAWSCloudFrontDistribution_validateHTTP(t *testing.T) {
	var value string
	var errors []error

	value = "incorrect"
	_, errors = validateHTTP(value, "http_version")
	if len(errors) == 0 {
		t.Fatalf("Expected %q to trigger a validation error", value)
	}

	value = "http1.1"
	_, errors = validateHTTP(value, "http_version")
	if len(errors) != 0 {
		t.Fatalf("Expected %q not to trigger a validation error", value)
	}
}

func testAccCheckCloudFrontDistributionDestroy(s *terraform.State) error {
	for k, rs := range s.RootModule().Resources {
		if rs.Type != "aws_cloudfront_distribution" {
			continue
		}
		dist, err := testAccAuxCloudFrontGetDistributionConfig(s, k)
		if err == nil {
			if _, ok := os.LookupEnv("TF_TEST_CLOUDFRONT_RETAIN"); ok {
				if *dist.DistributionConfig.Enabled != false {
					return fmt.Errorf("CloudFront distribution should be disabled")
				}
				return nil
			}
			return fmt.Errorf("CloudFront distribution did not destroy")
		}
	}
	return nil
}

func testAccCheckCloudFrontDistributionExistence(cloudFrontResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, err := testAccAuxCloudFrontGetDistributionConfig(s, cloudFrontResource)

		return err
	}
}

func testAccAuxCloudFrontGetDistributionConfig(s *terraform.State, cloudFrontResource string) (*cloudfront.Distribution, error) {
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

func testAccAWSCloudFrontDistributionRetainConfig() string {
	if _, ok := os.LookupEnv("TF_TEST_CLOUDFRONT_RETAIN"); ok {
		return "retain_on_delete = true"
	}
	return ""
}

var originBucket = fmt.Sprintf(`
resource "aws_s3_bucket" "s3_bucket_origin" {
	bucket = "mybucket.${var.rand_id}"
	acl = "public-read"
}
`)

var logBucket = fmt.Sprintf(`
resource "aws_s3_bucket" "s3_bucket_logs" {
	bucket = "mylogs.${var.rand_id}"
	acl = "public-read"
}
`)

var testAccAWSCloudFrontDistributionS3Config = `
variable rand_id {
	default = %d
}

# origin bucket
%s

# log bucket
%s

resource "aws_cloudfront_distribution" "s3_distribution" {
	origin {
		domain_name = "${aws_s3_bucket.s3_bucket_origin.id}.s3.amazonaws.com"
		origin_id = "myS3Origin"
	}
	enabled = true
	default_root_object = "index.html"
	logging_config {
		include_cookies = false
		bucket = "${aws_s3_bucket.s3_bucket_logs.id}.s3.amazonaws.com"
		prefix = "myprefix"
	}
	aliases = [ "mysite.${var.rand_id}.example.com", "yoursite.${var.rand_id}.example.com" ]
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myS3Origin"
		forwarded_values {
			query_string = false
			cookies {
				forward = "none"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	price_class = "PriceClass_200"
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	%s
}
`

var testAccAWSCloudFrontDistributionS3ConfigWithTags = `
variable rand_id {
	default = %d
}

# origin bucket
%s

# log bucket
%s

resource "aws_cloudfront_distribution" "s3_distribution" {
	origin {
		domain_name = "${aws_s3_bucket.s3_bucket_origin.id}.s3.amazonaws.com"
		origin_id = "myS3Origin"
	}
	enabled = true
	default_root_object = "index.html"
	aliases = [ "mysite.${var.rand_id}.example.com", "yoursite.${var.rand_id}.example.com" ]
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myS3Origin"
		forwarded_values {
			query_string = false
			cookies {
				forward = "none"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	price_class = "PriceClass_200"
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	tags {
            environment = "production"
            account = "main"
	}
	%s
}
`

var testAccAWSCloudFrontDistributionS3ConfigWithTagsUpdated = `
variable rand_id {
	default = %d
}

# origin bucket
%s

# log bucket
%s

resource "aws_cloudfront_distribution" "s3_distribution" {
	origin {
		domain_name = "${aws_s3_bucket.s3_bucket_origin.id}.s3.amazonaws.com"
		origin_id = "myS3Origin"
	}
	enabled = true
	default_root_object = "index.html"
	aliases = [ "mysite.${var.rand_id}.example.com", "yoursite.${var.rand_id}.example.com" ]
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myS3Origin"
		forwarded_values {
			query_string = false
			cookies {
				forward = "none"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	price_class = "PriceClass_200"
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	tags {
            environment = "dev"
	}
	%s
}
`

var testAccAWSCloudFrontDistributionCustomConfig = fmt.Sprintf(`
variable rand_id {
	default = %d
}

# log bucket
%s

resource "aws_cloudfront_distribution" "custom_distribution" {
	origin {
		domain_name = "www.example.com"
		origin_id = "myCustomOrigin"
		custom_origin_config {
			http_port = 80
			https_port = 443
			origin_protocol_policy = "http-only"
			origin_ssl_protocols = [ "SSLv3", "TLSv1" ]
			origin_read_timeout = 30
			origin_keepalive_timeout = 5
		}
	}
	enabled = true
	comment = "Some comment"
	default_root_object = "index.html"
	logging_config {
		include_cookies = false
		bucket = "${aws_s3_bucket.s3_bucket_logs.id}.s3.amazonaws.com"
		prefix = "myprefix"
	}
	aliases = [ "mysite.${var.rand_id}.example.com", "*.yoursite.${var.rand_id}.example.com" ]
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myCustomOrigin"
		smooth_streaming = false
		forwarded_values {
			query_string = false
			cookies {
				forward = "all"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	price_class = "PriceClass_200"
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	%s
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int(), logBucket, testAccAWSCloudFrontDistributionRetainConfig())

var testAccAWSCloudFrontDistributionMultiOriginConfig = fmt.Sprintf(`
variable rand_id {
	default = %d
}

# origin bucket
%s

# log bucket
%s

resource "aws_cloudfront_distribution" "multi_origin_distribution" {
	origin {
		domain_name = "${aws_s3_bucket.s3_bucket_origin.id}.s3.amazonaws.com"
		origin_id = "myS3Origin"
	}
	origin {
		domain_name = "www.example.com"
		origin_id = "myCustomOrigin"
		custom_origin_config {
			http_port = 80
			https_port = 443
			origin_protocol_policy = "http-only"
			origin_ssl_protocols = [ "SSLv3", "TLSv1" ]
			origin_keepalive_timeout = 45
		}
	}
	enabled = true
	comment = "Some comment"
	default_root_object = "index.html"
	logging_config {
		include_cookies = false
		bucket = "${aws_s3_bucket.s3_bucket_logs.id}.s3.amazonaws.com"
		prefix = "myprefix"
	}
	aliases = [ "mysite.${var.rand_id}.example.com", "*.yoursite.${var.rand_id}.example.com" ]
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myS3Origin"
		smooth_streaming = true
		forwarded_values {
			query_string = false
			cookies {
				forward = "all"
			}
		}
		min_ttl = 100
		default_ttl = 100
		max_ttl = 100
		viewer_protocol_policy = "allow-all"
	}
	cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myS3Origin"
		forwarded_values {
			query_string = true
			cookies {
				forward = "none"
			}
		}
		min_ttl = 50
		default_ttl = 50
		max_ttl = 50
		viewer_protocol_policy = "allow-all"
		path_pattern = "images1/*.jpg"
	}
	cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myCustomOrigin"
		forwarded_values {
			query_string = true
			cookies {
				forward = "none"
			}
		}
		min_ttl = 50
		default_ttl = 50
		max_ttl = 50
		viewer_protocol_policy = "allow-all"
		path_pattern = "images2/*.jpg"
	}

	price_class = "PriceClass_All"
	custom_error_response {
		error_code = 404
		response_page_path = "/error-pages/404.html"
		response_code = 200
		error_caching_min_ttl = 30
	}
	restrictions {
		geo_restriction {
			restriction_type = "none"
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	%s
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int(), originBucket, logBucket, testAccAWSCloudFrontDistributionRetainConfig())

var testAccAWSCloudFrontDistributionNoCustomErroResponseInfo = fmt.Sprintf(`
variable rand_id {
	default = %d
}

resource "aws_cloudfront_distribution" "no_custom_error_responses" {
	origin {
		domain_name = "www.example.com"
		origin_id = "myCustomOrigin"
		custom_origin_config {
			http_port = 80
			https_port = 443
			origin_protocol_policy = "http-only"
			origin_ssl_protocols = [ "SSLv3", "TLSv1" ]
		}
	}
	enabled = true
	comment = "Some comment"
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myCustomOrigin"
		smooth_streaming = false
		forwarded_values {
			query_string = false
			cookies {
				forward = "all"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	custom_error_response {
		error_code = 404
		error_caching_min_ttl = 30
	}
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	%s
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int(), testAccAWSCloudFrontDistributionRetainConfig())

var testAccAWSCloudFrontDistributionNoOptionalItemsConfig = fmt.Sprintf(`
variable rand_id {
	default = %d
}

resource "aws_cloudfront_distribution" "no_optional_items" {
	origin {
		domain_name = "www.example.com"
		origin_id = "myCustomOrigin"
		custom_origin_config {
			http_port = 80
			https_port = 443
			origin_protocol_policy = "http-only"
			origin_ssl_protocols = [ "SSLv3", "TLSv1" ]
		}
	}
	enabled = true
	comment = "Some comment"
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myCustomOrigin"
		smooth_streaming = false
		forwarded_values {
			query_string = false
			cookies {
				forward = "all"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	%s
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int(), testAccAWSCloudFrontDistributionRetainConfig())

var testAccAWSCloudFrontDistributionHTTP11Config = fmt.Sprintf(`
variable rand_id {
	default = %d
}

resource "aws_cloudfront_distribution" "http_1_1" {
	origin {
		domain_name = "www.example.com"
		origin_id = "myCustomOrigin"
		custom_origin_config {
			http_port = 80
			https_port = 443
			origin_protocol_policy = "http-only"
			origin_ssl_protocols = [ "SSLv3", "TLSv1" ]
		}
	}
	enabled = true
	comment = "Some comment"
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myCustomOrigin"
		smooth_streaming = false
		forwarded_values {
			query_string = false
			cookies {
				forward = "all"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	http_version = "http1.1"
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	%s
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int(), testAccAWSCloudFrontDistributionRetainConfig())

var testAccAWSCloudFrontDistributionIsIPV6EnabledConfig = fmt.Sprintf(`
variable rand_id {
	default = %d
}

resource "aws_cloudfront_distribution" "is_ipv6_enabled" {
	origin {
		domain_name = "www.example.com"
		origin_id = "myCustomOrigin"
		custom_origin_config {
			http_port = 80
			https_port = 443
			origin_protocol_policy = "http-only"
			origin_ssl_protocols = [ "SSLv3", "TLSv1" ]
		}
	}
	enabled = true
	is_ipv6_enabled = true
	comment = "Some comment"
	default_cache_behavior {
		allowed_methods = [ "DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT" ]
		cached_methods = [ "GET", "HEAD" ]
		target_origin_id = "myCustomOrigin"
		smooth_streaming = false
		forwarded_values {
			query_string = false
			cookies {
				forward = "all"
			}
		}
		viewer_protocol_policy = "allow-all"
		min_ttl = 0
		default_ttl = 3600
		max_ttl = 86400
	}
	http_version = "http1.1"
	restrictions {
		geo_restriction {
			restriction_type = "whitelist"
			locations = [ "US", "CA", "GB", "DE" ]
		}
	}
	viewer_certificate {
		cloudfront_default_certificate = true
	}
	%s
}
`, rand.New(rand.NewSource(time.Now().UnixNano())).Int(), testAccAWSCloudFrontDistributionRetainConfig())
