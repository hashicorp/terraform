package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/emr"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"log"
	"testing"
)

func TestAccAWSEmrTaskGroup_basic(t *testing.T) {
	var jobFlow emr.RunJobFlowOutput
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSEmrTaskGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSEmrTaskGroupConfig,
				Check:  testAccCheckAWSEmrTaskGroupExists("aws_emr_task_group.task", &jobFlow),
			},
		},
	})
}

func testAccCheckAWSEmrTaskGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).emrconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_emr" {
			continue
		}

		params := &emr.DescribeClusterInput{
			ClusterId: aws.String(rs.Primary.ID),
		}

		describe, err := conn.DescribeCluster(params)

		if err == nil {
			if describe.Cluster != nil &&
				*describe.Cluster.Status.State == "WAITING" {
				return fmt.Errorf("EMR Cluster still exists")
			}
		}

		providerErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		log.Printf("[ERROR] %v", providerErr)
	}

	return nil
}

func testAccCheckAWSEmrTaskGroupExists(n string, v *emr.RunJobFlowOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No task group id set")
		}
		conn := testAccProvider.Meta().(*AWSClient).emrconn
		_, err := conn.DescribeCluster(&emr.DescribeClusterInput{
			ClusterId: aws.String(rs.Primary.Attributes["cluster_id"]),
		})
		if err != nil {
			return fmt.Errorf("EMR error: %v", err)
		}
		return nil
	}
}

var testAccAWSEmrTaskGroupConfig = fmt.Sprintf(`
provider "aws" {
   region = "ap-southeast-2"
}

resource "aws_emr" "tf-test-cluster" {
  name          = "emr-%s"
  release_label = "emr-4.6.0"
  applications  = ["Spark"]

  master_instance_type = "m3.xlarge"
  core_instance_type   = "m3.xlarge"
  core_instance_count  = 1

  tags {
        role        = "rolename"
        dns_zone    = "env_zone"
        env         = "env"
        name        = "name-env"
  }

  bootstrap_action {
    path  ="s3://elasticmapreduce/bootstrap-actions/run-if"
    name  ="runif"
    args  =["instance.isMaster=true","echo running on master node"]
  }

  configurations = "test-fixtures/emr_configurations.json"
}

resource "aws_emr_task_group" "task" {
  cluster_id     = "${aws_emr.tf-test-cluster.id}"
  instance_count = 1
  instance_type  = "m3.xlarge"
}
`, acctest.RandString(10))
