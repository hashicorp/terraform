package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsDmsEndpointBasic(t *testing.T) {
	resourceName := "aws_dms_endpoint.dms_endpoint"
	randId := acctest.RandString(8) + "-basic"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsEndpointBasicConfig(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsEndpointExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "endpoint_arn"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: dmsEndpointBasicConfigUpdate(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsEndpointExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "database_name", "tf-test-dms-db-updated"),
					resource.TestCheckResourceAttr(resourceName, "extra_connection_attributes", "extra"),
					resource.TestCheckResourceAttr(resourceName, "password", "tftestupdate"),
					resource.TestCheckResourceAttr(resourceName, "port", "3303"),
					resource.TestCheckResourceAttr(resourceName, "ssl_mode", "none"),
					resource.TestCheckResourceAttr(resourceName, "server_name", "tftestupdate"),
					resource.TestCheckResourceAttr(resourceName, "username", "tftestupdate"),
				),
			},
		},
	})
}

func TestAccAwsDmsEndpointDynamoDb(t *testing.T) {
	resourceName := "aws_dms_endpoint.dms_endpoint"
	randId := acctest.RandString(8) + "-dynamodb"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsEndpointDynamoDbConfig(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsEndpointExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "endpoint_arn"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: dmsEndpointDynamoDbConfigUpdate(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsEndpointExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "ssl_mode", "none"),
					resource.TestCheckResourceAttr(resourceName, "server_name", "tftestupdate"),
				),
			},
		},
	})
}

func dmsEndpointDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dms_endpoint" {
			continue
		}

		err := checkDmsEndpointExists(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Found an endpoint that was not destroyed: %s", rs.Primary.ID)
		}
	}

	return nil
}

func checkDmsEndpointExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).dmsconn
		resp, err := conn.DescribeEndpoints(&dms.DescribeEndpointsInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("endpoint-id"),
					Values: []*string{aws.String(rs.Primary.ID)},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("DMS endpoint error: %v", err)
		}

		if resp.Endpoints == nil {
			return fmt.Errorf("DMS endpoint not found")
		}

		return nil
	}
}

func dmsEndpointBasicConfig(randId string) string {
	return fmt.Sprintf(`
resource "aws_dms_endpoint" "dms_endpoint" {
	database_name = "tf-test-dms-db"
	endpoint_id = "tf-test-dms-endpoint-%[1]s"
	endpoint_type = "source"
	engine_name = "aurora"
	extra_connection_attributes = ""
	password = "tftest"
	port = 3306
	server_name = "tftest"
	ssl_mode = "none"
	tags {
		Name = "tf-test-dms-endpoint-%[1]s"
		Update = "to-update"
		Remove = "to-remove"
	}
	username = "tftest"
}
`, randId)
}

func dmsEndpointBasicConfigUpdate(randId string) string {
	return fmt.Sprintf(`
resource "aws_dms_endpoint" "dms_endpoint" {
	database_name = "tf-test-dms-db-updated"
	endpoint_id = "tf-test-dms-endpoint-%[1]s"
	endpoint_type = "source"
	engine_name = "aurora"
	extra_connection_attributes = "extra"
	password = "tftestupdate"
	port = 3303
	server_name = "tftestupdate"
	ssl_mode = "none"
	tags {
		Name = "tf-test-dms-endpoint-%[1]s"
		Update = "updated"
		Add = "added"
	}
	username = "tftestupdate"
}
`, randId)
}

func dmsEndpointDynamoDbConfig(randId string) string {
	return fmt.Sprintf(`
resource "aws_dms_endpoint" "dms_endpoint" {
	endpoint_id = "tf-test-dms-endpoint-%[1]s"
	endpoint_type = "target"
	engine_name = "dynamodb"
	server_name = "tftest"
	service_access_role = "${aws_iam_role.iam_role.arn}"
	ssl_mode = "none"
	tags {
		Name = "tf-test-dynamodb-endpoint-%[1]s"
		Update = "to-update"
		Remove = "to-remove"
	}
}
resource "aws_iam_role" "iam_role" {
  name = "tf-test-iam-dynamodb-role-%[1]s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "dms.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "dms_dynamodb_access" {
  name = "tf-test-iam-dynamodb-role-policy-%[1]s"
  role = "${aws_iam_role.iam_role.name}"

  policy = <<EOF
{
"Version": "2012-10-17",
"Statement": [
{
    "Effect": "Allow",
    "Action": [
        "dynamodb:PutItem",
        "dynamodb:CreateTable",
        "dynamodb:DescribeTable",
        "dynamodb:DeleteTable",
        "dynamodb:DeleteItem",
        "dynamodb:ListTables"
    ],
    "Resource": "*"
}
]
}
EOF
}

`, randId)
}

func dmsEndpointDynamoDbConfigUpdate(randId string) string {
	return fmt.Sprintf(`
resource "aws_dms_endpoint" "dms_endpoint" {
	endpoint_id = "tf-test-dms-endpoint-%[1]s"
	endpoint_type = "target"
	engine_name = "dynamodb"
	server_name = "tftestupdate"
	service_access_role = "${aws_iam_role.iam_role.arn}"
	ssl_mode = "none"
	tags {
		Name = "tf-test-dynamodb-endpoint-%[1]s"
		Update = "updated"
		Add = "added"
	}
}
resource "aws_iam_role" "iam_role" {
  name = "tf-test-iam-dynamodb-role-%[1]s"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "dms.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "dms_dynamodb_access" {
  name = "tf-test-iam-dynamodb-role-policy-%[1]s"
  role = "${aws_iam_role.iam_role.name}"

  policy = <<EOF
{
"Version": "2012-10-17",
"Statement": [
{
    "Effect": "Allow",
    "Action": [
        "dynamodb:PutItem",
        "dynamodb:CreateTable",
        "dynamodb:DescribeTable",
        "dynamodb:DeleteTable",
        "dynamodb:DeleteItem",
        "dynamodb:ListTables"
    ],
    "Resource": "*"
}
]
}
EOF
}
`, randId)
}
