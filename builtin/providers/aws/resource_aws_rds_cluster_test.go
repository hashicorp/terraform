package aws

import (
        "fmt"
        "math/rand"
        "testing"
        "time"

        "github.com/hashicorp/terraform/helper/resource"
        "github.com/hashicorp/terraform/terraform"

        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-sdk-go/service/rds"
)

func TestAccAWSRDSCluster_basic(t *testing.T) {
        var v rds.DBCluster

        resource.Test(t, resource.TestCase{
                PreCheck:     func() { testAccPreCheck(t) },
                Providers:    testAccProviders,
                CheckDestroy: testAccCheckAWSClusterDestroy,
                Steps: []resource.TestStep{
                        resource.TestStep{
                                Config: testAccAWSClusterConfig,
                                Check: resource.ComposeTestCheckFunc(
                                        testAccCheckAWSClusterExists("aws_rds_cluster.default", &v),
                                ),
                        },
                },
        })
}

func testAccCheckAWSClusterDestroy(s *terraform.State) error {
        for _, rs := range s.RootModule().Resources {
                if rs.Type != "aws_rds_cluster" {
                        continue
                }

                // Try to find the Group
                conn := testAccProvider.Meta().(*AWSClient).rdsconn
                var err error
                resp, err := conn.DescribeDBClusters(
                        &rds.DescribeDBClustersInput{
                                DBClusterIdentifier: aws.String(rs.Primary.ID),
                        })

                if err == nil {
                        if len(resp.DBClusters) != 0 &&
                                *resp.DBClusters[0].DBClusterIdentifier == rs.Primary.ID {
                                return fmt.Errorf("DB Cluster %s still exists", rs.Primary.ID)
                        }
                }

                // check for an expected "Cluster not found" type error
                return err

        }

        return nil
}

func testAccCheckAWSClusterExists(n string, v *rds.DBCluster) resource.TestCheckFunc {
        return func(s *terraform.State) error {
                rs, ok := s.RootModule().Resources[n]
                if !ok {
                        return fmt.Errorf("Not found: %s", n)
                }

                if rs.Primary.ID == "" {
                        return fmt.Errorf("No DB Instance ID is set")
                }

                conn := testAccProvider.Meta().(*AWSClient).rdsconn
                resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
                        DBClusterIdentifier: aws.String(rs.Primary.ID),
                })

                if err != nil {
                        return err
                }

                for _, c := range resp.DBClusters {
                        if *c.DBClusterIdentifier == rs.Primary.ID {
                                *v = *c
                                return nil
                        }
                }

                return fmt.Errorf("DB Cluster (%s) not found", rs.Primary.ID)
        }
}

// Add some random to the name, to avoid collision
var testAccAWSClusterConfig = fmt.Sprintf(`
resource "aws_rds_cluster" "default" {
  cluster_identifier = "tf-aurora-cluster-%d"
  availability_zones = ["us-west-2a","us-west-2b","us-west-2c"]
  database_name = "mydb"
  master_username = "foo"
  master_password = "mustbeeightcharaters"
}`, rand.New(rand.NewSource(time.Now().UnixNano())).Int())
