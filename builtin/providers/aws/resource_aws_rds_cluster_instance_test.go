package aws

import (
        "fmt"
        "math/rand"
        "strings"
        "testing"
        "time"

        "github.com/hashicorp/terraform/helper/resource"
        "github.com/hashicorp/terraform/terraform"

        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-sdk-go/aws/awserr"
        "github.com/aws/aws-sdk-go/service/rds"
)

func TestAccAWSRDSClusterInstance_basic(t *testing.T) {
        var v rds.DBInstance

        resource.Test(t, resource.TestCase{
                PreCheck:     func() { testAccPreCheck(t) },
                Providers:    testAccProviders,
                CheckDestroy: testAccCheckAWSClusterDestroy,
                Steps: []resource.TestStep{
                        resource.TestStep{
                                Config: testAccAWSClusterInstanceConfig,
                                Check: resource.ComposeTestCheckFunc(
                                        testAccCheckAWSClusterInstanceExists("aws_rds_cluster_instance.cluster_instances", &v),
                                        testAccCheckAWSDBClusterInstanceAttributes(&v),
                                ),
                        },
                },
        })
}

func testAccCheckAWSClusterInstanceDestroy(s *terraform.State) error {
        for _, rs := range s.RootModule().Resources {
                if rs.Type != "aws_rds_cluster" {
                        continue
                }

                // Try to find the Group
                conn := testAccProvider.Meta().(*AWSClient).rdsconn
                var err error
                resp, err := conn.DescribeDBInstances(
                        &rds.DescribeDBInstancesInput{
                                DBInstanceIdentifier: aws.String(rs.Primary.ID),
                        })

                if err == nil {
                        if len(resp.DBInstances) != 0 &&
                                *resp.DBInstances[0].DBInstanceIdentifier == rs.Primary.ID {
                                return fmt.Errorf("DB Cluster Instance %s still exists", rs.Primary.ID)
                        }
                }

                // Return nil if the Cluster Instance is already destroyed
                if awsErr, ok := err.(awserr.Error); ok {
                        if awsErr.Code() == "DBInstanceNotFound" {
                                return nil
                        }
                }

                return err

        }

        return nil
}

func testAccCheckAWSDBClusterInstanceAttributes(v *rds.DBInstance) resource.TestCheckFunc {
        return func(s *terraform.State) error {

                if *v.Engine != "aurora" {
                        return fmt.Errorf("bad engine, expected \"aurora\": %#v", *v.Engine)
                }

                if !strings.HasPrefix(*v.DBClusterIdentifier, "tf-aurora-cluster") {
                        return fmt.Errorf("Bad Cluster Identifier prefix:\nexpected: %s\ngot: %s", "tf-aurora-cluster", *v.DBClusterIdentifier)
                }

                return nil
        }
}

func testAccCheckAWSClusterInstanceExists(n string, v *rds.DBInstance) resource.TestCheckFunc {
        return func(s *terraform.State) error {
                rs, ok := s.RootModule().Resources[n]
                if !ok {
                        return fmt.Errorf("Not found: %s", n)
                }

                if rs.Primary.ID == "" {
                        return fmt.Errorf("No DB Instance ID is set")
                }

                conn := testAccProvider.Meta().(*AWSClient).rdsconn
                resp, err := conn.DescribeDBInstances(&rds.DescribeDBInstancesInput{
                        DBInstanceIdentifier: aws.String(rs.Primary.ID),
                })

                if err != nil {
                        return err
                }

                for _, d := range resp.DBInstances {
                        if *d.DBInstanceIdentifier == rs.Primary.ID {
                                *v = *d
                                return nil
                        }
                }

                return fmt.Errorf("DB Cluster (%s) not found", rs.Primary.ID)
        }
}

// Add some random to the name, to avoid collision
var testAccAWSClusterInstanceConfig = fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-test-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
}

resource "aws_rds_cluster_instance" "cluster_instances" {
  identifier = "aurora-cluster-test-instance"
        cluster_identifier = "${aws_rds_cluster.default.id}"
  instance_class = "db.r3.large"
}

`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
