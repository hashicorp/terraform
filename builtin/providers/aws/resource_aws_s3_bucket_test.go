package aws

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestAccAWSS3Bucket_basic(t *testing.T) {
	rInt := acctest.RandInt()
	arnRegexp := regexp.MustCompile(
		"^arn:aws:s3:::")

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		/*
			IDRefreshName:   "aws_s3_bucket.bucket",
			IDRefreshIgnore: []string{"force_destroy"},
		*/
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "hosted_zone_id", HostedZoneIDForRegion("us-west-2")),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "region", "us-west-2"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", ""),
					resource.TestMatchResourceAttr(
						"aws_s3_bucket.bucket", "arn", arnRegexp),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_Policy(t *testing.T) {
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithPolicy(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketPolicy(
						"aws_s3_bucket.bucket", testAccAWSS3BucketPolicy(rInt)),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketPolicy(
						"aws_s3_bucket.bucket", ""),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithEmptyPolicy(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketPolicy(
						"aws_s3_bucket.bucket", ""),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_UpdateAcl(t *testing.T) {
	ri := acctest.RandInt()
	preConfig := fmt.Sprintf(testAccAWSS3BucketConfigWithAcl, ri)
	postConfig := fmt.Sprintf(testAccAWSS3BucketConfigWithAclUpdate, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: preConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "acl", "public-read"),
				),
			},
			resource.TestStep{
				Config: postConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "acl", "private"),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_Website_Simple(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketWebsiteConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "index.html", "", "", ""),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", testAccWebsiteEndpoint(rInt)),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketWebsiteConfigWithError(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "index.html", "error.html", "", ""),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", testAccWebsiteEndpoint(rInt)),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "", "", "", ""),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", ""),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_WebsiteRedirect(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketWebsiteConfigWithRedirect(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "", "", "", "hashicorp.com"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", testAccWebsiteEndpoint(rInt)),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketWebsiteConfigWithHttpsRedirect(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "", "", "https", "hashicorp.com"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", testAccWebsiteEndpoint(rInt)),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "", "", "", ""),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", ""),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_WebsiteRoutingRules(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketWebsiteConfigWithRoutingRules(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "index.html", "error.html", "", ""),
					testAccCheckAWSS3BucketWebsiteRoutingRules(
						"aws_s3_bucket.bucket",
						[]*s3.RoutingRule{
							&s3.RoutingRule{
								Condition: &s3.Condition{
									KeyPrefixEquals: aws.String("docs/"),
								},
								Redirect: &s3.Redirect{
									ReplaceKeyPrefixWith: aws.String("documents/"),
								},
							},
						},
					),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", testAccWebsiteEndpoint(rInt)),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketWebsite(
						"aws_s3_bucket.bucket", "", "", "", ""),
					testAccCheckAWSS3BucketWebsiteRoutingRules("aws_s3_bucket.bucket", nil),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "website_endpoint", ""),
				),
			},
		},
	})
}

// Test TestAccAWSS3Bucket_shouldFailNotFound is designed to fail with a "plan
// not empty" error in Terraform, to check against regresssions.
// See https://github.com/hashicorp/terraform/pull/2925
func TestAccAWSS3Bucket_shouldFailNotFound(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketDestroyedConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3DestroyBucket("aws_s3_bucket.bucket"),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAWSS3Bucket_Versioning(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketVersioning(
						"aws_s3_bucket.bucket", ""),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithVersioning(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketVersioning(
						"aws_s3_bucket.bucket", s3.BucketVersioningStatusEnabled),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithDisableVersioning(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketVersioning(
						"aws_s3_bucket.bucket", s3.BucketVersioningStatusSuspended),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_Cors(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithCORS(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketCors(
						"aws_s3_bucket.bucket",
						[]*s3.CORSRule{
							&s3.CORSRule{
								AllowedHeaders: []*string{aws.String("*")},
								AllowedMethods: []*string{aws.String("PUT"), aws.String("POST")},
								AllowedOrigins: []*string{aws.String("https://www.example.com")},
								ExposeHeaders:  []*string{aws.String("x-amz-server-side-encryption"), aws.String("ETag")},
								MaxAgeSeconds:  aws.Int64(3000),
							},
						},
					),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_Logging(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithLogging(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					testAccCheckAWSS3BucketLogging(
						"aws_s3_bucket.bucket", "aws_s3_bucket.log_bucket", "log/"),
				),
			},
		},
	})
}

func TestAccAWSS3Bucket_Lifecycle(t *testing.T) {
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSS3BucketDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithLifecycle(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.id", "id1"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.prefix", "path1/"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.expiration.2613713285.days", "365"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.expiration.2613713285.date", ""),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.expiration.2613713285.expired_object_delete_marker", "false"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.transition.2000431762.date", ""),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.transition.2000431762.days", "30"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.transition.2000431762.storage_class", "STANDARD_IA"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.transition.6450812.date", ""),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.transition.6450812.days", "60"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.transition.6450812.storage_class", "GLACIER"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.id", "id2"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.prefix", "path2/"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.expiration.2855832418.date", "2016-01-12"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.expiration.2855832418.days", "0"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.expiration.2855832418.expired_object_delete_marker", "false"),
				),
			},
			resource.TestStep{
				Config: testAccAWSS3BucketConfigWithVersioningLifecycle(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSS3BucketExists("aws_s3_bucket.bucket"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.id", "id1"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.prefix", "path1/"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.enabled", "true"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.noncurrent_version_expiration.80908210.days", "365"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.noncurrent_version_transition.1377917700.days", "30"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.noncurrent_version_transition.1377917700.storage_class", "STANDARD_IA"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.noncurrent_version_transition.2528035817.days", "60"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.0.noncurrent_version_transition.2528035817.storage_class", "GLACIER"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.id", "id2"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.prefix", "path2/"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.enabled", "false"),
					resource.TestCheckResourceAttr(
						"aws_s3_bucket.bucket", "lifecycle_rule.1.noncurrent_version_expiration.80908210.days", "365"),
				),
			},
		},
	})
}

func testAccCheckAWSS3BucketDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).s3conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_s3_bucket" {
			continue
		}
		_, err := conn.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: aws.String(rs.Primary.ID),
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchBucket" {
				return nil
			}
			return err
		}
	}
	return nil
}

func testAccCheckAWSS3BucketExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No S3 Bucket ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).s3conn
		_, err := conn.HeadBucket(&s3.HeadBucketInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("S3Bucket error: %v", err)
		}
		return nil
	}
}

func testAccCheckAWSS3DestroyBucket(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No S3 Bucket ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).s3conn
		_, err := conn.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("Error destroying Bucket (%s) in testAccCheckAWSS3DestroyBucket: %s", rs.Primary.ID, err)
		}
		return nil
	}
}

func testAccCheckAWSS3BucketPolicy(n string, policy string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := conn.GetBucketPolicy(&s3.GetBucketPolicyInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			if policy == "" {
				// expected
				return nil
			} else {
				return fmt.Errorf("GetBucketPolicy error: %v, expected %s", err, policy)
			}
		}

		if v := out.Policy; v == nil {
			if policy != "" {
				return fmt.Errorf("bad policy, found nil, expected: %s", policy)
			}
		} else {
			expected := make(map[string]interface{})
			if err := json.Unmarshal([]byte(policy), &expected); err != nil {
				return err
			}
			actual := make(map[string]interface{})
			if err := json.Unmarshal([]byte(*v), &actual); err != nil {
				return err
			}

			if !reflect.DeepEqual(expected, actual) {
				return fmt.Errorf("bad policy, expected: %#v, got %#v", expected, actual)
			}
		}

		return nil
	}
}

func testAccCheckAWSS3BucketWebsite(n string, indexDoc string, errorDoc string, redirectProtocol string, redirectTo string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := conn.GetBucketWebsite(&s3.GetBucketWebsiteInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			if indexDoc == "" {
				// If we want to assert that the website is not there, than
				// this error is expected
				return nil
			} else {
				return fmt.Errorf("S3BucketWebsite error: %v", err)
			}
		}

		if v := out.IndexDocument; v == nil {
			if indexDoc != "" {
				return fmt.Errorf("bad index doc, found nil, expected: %s", indexDoc)
			}
		} else {
			if *v.Suffix != indexDoc {
				return fmt.Errorf("bad index doc, expected: %s, got %#v", indexDoc, out.IndexDocument)
			}
		}

		if v := out.ErrorDocument; v == nil {
			if errorDoc != "" {
				return fmt.Errorf("bad error doc, found nil, expected: %s", errorDoc)
			}
		} else {
			if *v.Key != errorDoc {
				return fmt.Errorf("bad error doc, expected: %s, got %#v", errorDoc, out.ErrorDocument)
			}
		}

		if v := out.RedirectAllRequestsTo; v == nil {
			if redirectTo != "" {
				return fmt.Errorf("bad redirect to, found nil, expected: %s", redirectTo)
			}
		} else {
			if *v.HostName != redirectTo {
				return fmt.Errorf("bad redirect to, expected: %s, got %#v", redirectTo, out.RedirectAllRequestsTo)
			}
			if redirectProtocol != "" && v.Protocol != nil && *v.Protocol != redirectProtocol {
				return fmt.Errorf("bad redirect protocol to, expected: %s, got %#v", redirectProtocol, out.RedirectAllRequestsTo)
			}
		}

		return nil
	}
}

func testAccCheckAWSS3BucketWebsiteRoutingRules(n string, routingRules []*s3.RoutingRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := conn.GetBucketWebsite(&s3.GetBucketWebsiteInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			if routingRules == nil {
				return nil
			}
			return fmt.Errorf("GetBucketWebsite error: %v", err)
		}

		if !reflect.DeepEqual(out.RoutingRules, routingRules) {
			return fmt.Errorf("bad routing rule, expected: %v, got %v", routingRules, out.RoutingRules)
		}

		return nil
	}
}

func testAccCheckAWSS3BucketVersioning(n string, versioningStatus string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := conn.GetBucketVersioning(&s3.GetBucketVersioningInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("GetBucketVersioning error: %v", err)
		}

		if v := out.Status; v == nil {
			if versioningStatus != "" {
				return fmt.Errorf("bad error versioning status, found nil, expected: %s", versioningStatus)
			}
		} else {
			if *v != versioningStatus {
				return fmt.Errorf("bad error versioning status, expected: %s, got %s", versioningStatus, *v)
			}
		}

		return nil
	}
}

func testAccCheckAWSS3BucketCors(n string, corsRules []*s3.CORSRule) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := conn.GetBucketCors(&s3.GetBucketCorsInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("GetBucketCors error: %v", err)
		}

		if !reflect.DeepEqual(out.CORSRules, corsRules) {
			return fmt.Errorf("bad error cors rule, expected: %v, got %v", corsRules, out.CORSRules)
		}

		return nil
	}
}

func testAccCheckAWSS3BucketLogging(n, b, p string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, _ := s.RootModule().Resources[n]
		conn := testAccProvider.Meta().(*AWSClient).s3conn

		out, err := conn.GetBucketLogging(&s3.GetBucketLoggingInput{
			Bucket: aws.String(rs.Primary.ID),
		})

		if err != nil {
			return fmt.Errorf("GetBucketLogging error: %v", err)
		}

		tb, _ := s.RootModule().Resources[b]

		if v := out.LoggingEnabled.TargetBucket; v == nil {
			if tb.Primary.ID != "" {
				return fmt.Errorf("bad target bucket, found nil, expected: %s", tb.Primary.ID)
			}
		} else {
			if *v != tb.Primary.ID {
				return fmt.Errorf("bad target bucket, expected: %s, got %s", tb.Primary.ID, *v)
			}
		}

		if v := out.LoggingEnabled.TargetPrefix; v == nil {
			if p != "" {
				return fmt.Errorf("bad target prefix, found nil, expected: %s", p)
			}
		} else {
			if *v != p {
				return fmt.Errorf("bad target prefix, expected: %s, got %s", p, *v)
			}
		}

		return nil
	}
}

// These need a bit of randomness as the name can only be used once globally
// within AWS
func testAccWebsiteEndpoint(randInt int) string {
	return fmt.Sprintf("tf-test-bucket-%d.s3-website-us-west-2.amazonaws.com", randInt)
}

func testAccAWSS3BucketPolicy(randInt int) string {
	return fmt.Sprintf(`{ "Version": "2012-10-17", "Statement": [ { "Sid": "", "Effect": "Allow", "Principal": { "AWS": "*" }, "Action": "s3:GetObject", "Resource": "arn:aws:s3:::tf-test-bucket-%d/*" } ] }`, randInt)
}

func testAccAWSS3BucketConfig(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
}
`, randInt)
}

func testAccAWSS3BucketWebsiteConfig(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"

	website {
		index_document = "index.html"
	}
}
`, randInt)
}

func testAccAWSS3BucketWebsiteConfigWithError(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"

	website {
		index_document = "index.html"
		error_document = "error.html"
	}
}
`, randInt)
}

func testAccAWSS3BucketWebsiteConfigWithRedirect(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"

	website {
		redirect_all_requests_to = "hashicorp.com"
	}
}
`, randInt)
}

func testAccAWSS3BucketWebsiteConfigWithHttpsRedirect(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"

	website {
		redirect_all_requests_to = "https://hashicorp.com"
	}
}
`, randInt)
}

func testAccAWSS3BucketWebsiteConfigWithRoutingRules(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"

	website {
		index_document = "index.html"
		error_document = "error.html"
		routing_rules = <<EOF
[{
	"Condition": {
		"KeyPrefixEquals": "docs/"
	},
	"Redirect": {
		"ReplaceKeyPrefixWith": "documents/"
	}
}]
EOF
	}
}
`, randInt)
}

func testAccAWSS3BucketConfigWithPolicy(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
	policy = %s
}
`, randInt, strconv.Quote(testAccAWSS3BucketPolicy(randInt)))
}

func testAccAWSS3BucketDestroyedConfig(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
}
`, randInt)
}

func testAccAWSS3BucketConfigWithEmptyPolicy(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
	policy = ""
}
`, randInt)
}

func testAccAWSS3BucketConfigWithVersioning(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
	versioning {
	  enabled = true
	}
}
`, randInt)
}

func testAccAWSS3BucketConfigWithDisableVersioning(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
	versioning {
	  enabled = false
	}
}
`, randInt)
}

func testAccAWSS3BucketConfigWithCORS(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
	cors_rule {
			allowed_headers = ["*"]
			allowed_methods = ["PUT","POST"]
			allowed_origins = ["https://www.example.com"]
			expose_headers = ["x-amz-server-side-encryption","ETag"]
			max_age_seconds = 3000
	}
}
`, randInt)
}

var testAccAWSS3BucketConfigWithAcl = `
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "public-read"
}
`

var testAccAWSS3BucketConfigWithAclUpdate = `
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
}
`

func testAccAWSS3BucketConfigWithLogging(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "log_bucket" {
	bucket = "tf-test-log-bucket-%d"
	acl = "log-delivery-write"
}
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
	logging {
		target_bucket = "${aws_s3_bucket.log_bucket.id}"
		target_prefix = "log/"
	}
}
`, randInt, randInt)
}

func testAccAWSS3BucketConfigWithLifecycle(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
	lifecycle_rule {
		id = "id1"
		prefix = "path1/"
		enabled = true

		expiration {
			days = 365
		}

		transition {
			days = 30
			storage_class = "STANDARD_IA"
		}
		transition {
			days = 60
			storage_class = "GLACIER"
		}
	}
	lifecycle_rule {
		id = "id2"
		prefix = "path2/"
		enabled = true

		expiration {
			date = "2016-01-12"
		}
	}
}
`, randInt)
}

func testAccAWSS3BucketConfigWithVersioningLifecycle(randInt int) string {
	return fmt.Sprintf(`
resource "aws_s3_bucket" "bucket" {
	bucket = "tf-test-bucket-%d"
	acl = "private"
	versioning {
	  enabled = false
	}
	lifecycle_rule {
		id = "id1"
		prefix = "path1/"
		enabled = true

		noncurrent_version_expiration {
			days = 365
		}
		noncurrent_version_transition {
			days = 30
			storage_class = "STANDARD_IA"
		}
		noncurrent_version_transition {
			days = 60
			storage_class = "GLACIER"
		}
	}
	lifecycle_rule {
		id = "id2"
		prefix = "path2/"
		enabled = false

		noncurrent_version_expiration {
			days = 365
		}
	}
}
`, randInt)
}
