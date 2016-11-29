package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticSearchDomain_basic(t *testing.T) {
	var domain elasticsearch.ElasticsearchDomainStatus
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckESDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccESDomainConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckESDomainExists("aws_elasticsearch_domain.example", &domain),
					resource.TestCheckResourceAttr(
						"aws_elasticsearch_domain.example", "elasticsearch_version", "1.5"),
				),
			},
		},
	})
}

func TestAccAWSElasticSearchDomain_v23(t *testing.T) {
	var domain elasticsearch.ElasticsearchDomainStatus
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckESDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccESDomainConfigV23(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckESDomainExists("aws_elasticsearch_domain.example", &domain),
					resource.TestCheckResourceAttr(
						"aws_elasticsearch_domain.example", "elasticsearch_version", "2.3"),
				),
			},
		},
	})
}

func TestAccAWSElasticSearchDomain_complex(t *testing.T) {
	var domain elasticsearch.ElasticsearchDomainStatus
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckESDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccESDomainConfig_complex(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckESDomainExists("aws_elasticsearch_domain.example", &domain),
				),
			},
		},
	})
}

func TestAccAWSElasticSearch_tags(t *testing.T) {
	var domain elasticsearch.ElasticsearchDomainStatus
	var td elasticsearch.ListTagsOutput
	ri := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSELBDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccESDomainConfig(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckESDomainExists("aws_elasticsearch_domain.example", &domain),
				),
			},

			resource.TestStep{
				Config: testAccESDomainConfig_TagUpdate(ri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckESDomainExists("aws_elasticsearch_domain.example", &domain),
					testAccLoadESTags(&domain, &td),
					testAccCheckElasticsearchServiceTags(&td.TagList, "foo", "bar"),
					testAccCheckElasticsearchServiceTags(&td.TagList, "new", "type"),
				),
			},
		},
	})
}

func testAccLoadESTags(conf *elasticsearch.ElasticsearchDomainStatus, td *elasticsearch.ListTagsOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := testAccProvider.Meta().(*AWSClient).esconn

		describe, err := conn.ListTags(&elasticsearch.ListTagsInput{
			ARN: conf.ARN,
		})

		if err != nil {
			return err
		}
		if len(describe.TagList) > 0 {
			*td = *describe
		}
		return nil
	}
}

func testAccCheckESDomainExists(n string, domain *elasticsearch.ElasticsearchDomainStatus) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ES Domain ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).esconn
		opts := &elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(rs.Primary.Attributes["domain_name"]),
		}

		resp, err := conn.DescribeElasticsearchDomain(opts)
		if err != nil {
			return fmt.Errorf("Error describing domain: %s", err.Error())
		}

		*domain = *resp.DomainStatus

		return nil
	}
}

func testAccCheckESDomainDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_elasticsearch_domain" {
			continue
		}

		conn := testAccProvider.Meta().(*AWSClient).esconn
		opts := &elasticsearch.DescribeElasticsearchDomainInput{
			DomainName: aws.String(rs.Primary.Attributes["domain_name"]),
		}

		_, err := conn.DescribeElasticsearchDomain(opts)
		// Verify the error is what we want
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceNotFoundException" {
				continue
			}
			return err
		}
	}
	return nil
}

func testAccESDomainConfig(randInt int) string {
	return fmt.Sprintf(`
resource "aws_elasticsearch_domain" "example" {
  domain_name = "tf-test-%d"
}
`, randInt)
}

func testAccESDomainConfig_TagUpdate(randInt int) string {
	return fmt.Sprintf(`
resource "aws_elasticsearch_domain" "example" {
  domain_name = "tf-test-%d"

  tags {
    foo = "bar"
    new = "type"
  }
}
`, randInt)
}

func testAccESDomainConfig_complex(randInt int) string {
	return fmt.Sprintf(`
resource "aws_elasticsearch_domain" "example" {
  domain_name = "tf-test-%d"

  advanced_options {
    "indices.fielddata.cache.size" = 80
  }

  ebs_options {
    ebs_enabled = false
  }

  cluster_config {
    instance_count = 2
    zone_awareness_enabled = true
  }

  snapshot_options {
    automated_snapshot_start_hour = 23
  }

  tags {
    bar = "complex"
  }
}
`, randInt)
}

func testAccESDomainConfigV23(randInt int) string {
	return fmt.Sprintf(`
resource "aws_elasticsearch_domain" "example" {
  domain_name = "tf-test-%d"
  elasticsearch_version = "2.3"
}
`, randInt)
}
