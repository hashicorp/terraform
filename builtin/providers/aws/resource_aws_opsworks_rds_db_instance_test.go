package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSOpsworksRdsDbInstance(t *testing.T) {
	sName := fmt.Sprintf("test-db-instance-%d", acctest.RandInt())
	var opsdb opsworks.RdsDbInstance
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAwsOpsworksRdsDbDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsOpsworksRdsDbInstance(sName, "foo", "barbarbarbar"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksRdsDbExists(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", &opsdb),
					testAccCheckAWSOpsworksCreateRdsDbAttributes(&opsdb, "foo"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "foo",
					),
				),
			},
			{
				Config: testAccAwsOpsworksRdsDbInstance(sName, "bar", "barbarbarbar"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksRdsDbExists(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", &opsdb),
					testAccCheckAWSOpsworksCreateRdsDbAttributes(&opsdb, "bar"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "bar",
					),
				),
			},
			{
				Config: testAccAwsOpsworksRdsDbInstance(sName, "bar", "foofoofoofoofoo"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksRdsDbExists(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", &opsdb),
					testAccCheckAWSOpsworksCreateRdsDbAttributes(&opsdb, "bar"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "bar",
					),
				),
			},
			{
				Config: testAccAwsOpsworksRdsDbInstanceForceNew(sName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSOpsworksRdsDbExists(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", &opsdb),
					testAccCheckAWSOpsworksCreateRdsDbAttributes(&opsdb, "foo"),
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "foo",
					),
				),
			},
		},
	})
}

func testAccCheckAWSOpsworksRdsDbExists(
	n string, opsdb *opsworks.RdsDbInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		if _, ok := rs.Primary.Attributes["stack_id"]; !ok {
			return fmt.Errorf("Rds Db stack id is missing, should be set.")
		}

		conn := testAccProvider.Meta().(*AWSClient).opsworksconn

		params := &opsworks.DescribeRdsDbInstancesInput{
			StackId: aws.String(rs.Primary.Attributes["stack_id"]),
		}
		resp, err := conn.DescribeRdsDbInstances(params)

		if err != nil {
			return err
		}

		if v := len(resp.RdsDbInstances); v != 1 {
			return fmt.Errorf("Expected 1 response returned, got %d", v)
		}

		*opsdb = *resp.RdsDbInstances[0]

		return nil
	}
}

func testAccCheckAWSOpsworksCreateRdsDbAttributes(
	opsdb *opsworks.RdsDbInstance, user string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *opsdb.DbUser != user {
			return fmt.Errorf("Unnexpected user: %s", *opsdb.DbUser)
		}
		if *opsdb.Engine != "mysql" {
			return fmt.Errorf("Unnexpected engine: %s", *opsdb.Engine)
		}
		return nil
	}
}

func testAccCheckAwsOpsworksRdsDbDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AWSClient).opsworksconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_opsworks_rds_db_instance" {
			continue
		}

		req := &opsworks.DescribeRdsDbInstancesInput{
			StackId: aws.String(rs.Primary.Attributes["stack_id"]),
		}

		resp, err := client.DescribeRdsDbInstances(req)
		if err == nil {
			if len(resp.RdsDbInstances) > 0 {
				return fmt.Errorf("OpsWorks Rds db instances  still exist.")
			}
		}

		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() != "ResourceNotFoundException" {
				return err
			}
		}
	}
	return nil
}

func testAccAwsOpsworksRdsDbInstance(name, userName, password string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_rds_db_instance" "tf-acc-opsworks-db" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  rds_db_instance_arn = "${aws_db_instance.bar.arn}"
  db_user             = "%s"
  db_password         = "%s"
}

%s

%s
`, userName, password, testAccAwsOpsworksStackConfigVpcCreate(name), testAccAWSDBInstanceConfig)
}

func testAccAwsOpsworksRdsDbInstanceForceNew(name string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_rds_db_instance" "tf-acc-opsworks-db" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  rds_db_instance_arn = "${aws_db_instance.foo.arn}"
  db_user             = "foo"
  db_password         = "foofoofoofoo"
}

%s

resource "aws_db_instance" "foo" {
  allocated_storage    = 10
  engine               = "MySQL"
  engine_version       = "5.6.21"
  instance_class       = "db.t1.micro"
  name                 = "baz"
  password             = "foofoofoofoo"
  username             = "foo"
  parameter_group_name = "default.mysql5.6"

  skip_final_snapshot = true
}
`, testAccAwsOpsworksStackConfigVpcCreate(name))
}
