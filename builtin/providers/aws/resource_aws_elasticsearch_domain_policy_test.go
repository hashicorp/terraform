package aws

import (
	"fmt"
	"testing"

	elasticsearch "github.com/aws/aws-sdk-go/service/elasticsearchservice"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSElasticSearchDomainPolicy_basic(t *testing.T) {
	var domain elasticsearch.ElasticsearchDomainStatus
	ri := acctest.RandInt()
	policy := `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "es:*",
            "Principal": "*",
            "Effect": "Allow",
            "Condition": {
                "IpAddress": {"aws:SourceIp": "127.0.0.1/32"}
            },
            "Resource": "${aws_elasticsearch_domain.example.arn}"
        }
    ]
}`
	expectedPolicyTpl := `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "es:*",
            "Principal": "*",
            "Effect": "Allow",
            "Condition": {
                "IpAddress": {"aws:SourceIp": "127.0.0.1/32"}
            },
            "Resource": "%s"
        }
    ]
}`
	name := fmt.Sprintf("tf-test-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckESDomainDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccESDomainPolicyConfig(ri, policy),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckESDomainExists("aws_elasticsearch_domain.example", &domain),
					resource.TestCheckResourceAttr("aws_elasticsearch_domain.example", "elasticsearch_version", "2.3"),
					func(s *terraform.State) error {
						awsClient := testAccProvider.Meta().(*AWSClient)
						expectedArn, err := buildESDomainArn(name, awsClient.partition, awsClient.accountid, awsClient.region)
						if err != nil {
							return err
						}
						expectedPolicy := fmt.Sprintf(expectedPolicyTpl, expectedArn)

						return testAccCheckAwsPolicyMatch("aws_elasticsearch_domain_policy.main", "access_policies", expectedPolicy)(s)
					},
				),
			},
		},
	})
}

func buildESDomainArn(name, partition, accId, region string) (string, error) {
	if partition == "" {
		return "", fmt.Errorf("Unable to construct ES Domain ARN because of missing AWS partition")
	}
	if accId == "" {
		return "", fmt.Errorf("Unable to construct ES Domain ARN because of missing AWS Account ID")
	}
	// arn:aws:es:us-west-2:187416307283:domain/example-name
	return fmt.Sprintf("arn:%s:es:%s:%s:domain/%s", partition, region, accId, name), nil
}

func testAccESDomainPolicyConfig(randInt int, policy string) string {
	return fmt.Sprintf(`
resource "aws_elasticsearch_domain" "example" {
    domain_name = "tf-test-%d"
    elasticsearch_version = "2.3"
    cluster_config {
        instance_type = "t2.micro.elasticsearch"
    }
    ebs_options {
        ebs_enabled = true
        volume_size = 10
    }
}

resource "aws_elasticsearch_domain_policy" "main" {
  domain_name = "${aws_elasticsearch_domain.example.domain_name}"
  access_policies = <<POLICIES
%s
POLICIES
}
`, randInt, policy)
}
