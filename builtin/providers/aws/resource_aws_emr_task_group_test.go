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

  ec2_attributes {
    subnet_id   = "${aws_subnet.main.id}"
    emr_managed_master_security_group = "${aws_security_group.allow_all.id}"
    emr_managed_slave_security_group = "${aws_security_group.allow_all.id}"
  }

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

resource "aws_security_group" "allow_all" {
  name        = "allow_all"
  description = "Allow all inbound traffic"
  vpc_id      = "${aws_vpc.main.id}"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }

  depends_on = ["aws_subnet.main"]
  lifecycle {
        ignore_changes = ["ingress", "egress"]
  }
}

resource "aws_vpc" "main" {
  cidr_block           = "168.31.0.0/16"
  enable_dns_hostnames = true
}

resource "aws_subnet" "main" {
  vpc_id                  = "${aws_vpc.main.id}"
  cidr_block              = "168.31.0.0/20"
#  map_public_ip_on_launch = true
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.main.id}"
}

resource "aws_route_table" "r" {
  vpc_id = "${aws_vpc.main.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }
}

resource "aws_main_route_table_association" "a" {
  vpc_id         = "${aws_vpc.main.id}"
  route_table_id = "${aws_route_table.r.id}"
}
`, acctest.RandString(10))
