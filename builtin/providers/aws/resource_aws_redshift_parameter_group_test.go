package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRedshiftParameterGroup_withParameters(t *testing.T) {
	var v redshift.ClusterParameterGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRedshiftParameterGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftParameterGroupExists("aws_redshift_parameter_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "name", "parameter-group-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "family", "redshift-1.0"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "description", "Test parameter group for terraform"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "parameter.490804664.name", "require_ssl"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "parameter.490804664.value", "true"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "parameter.2036118857.name", "query_group"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "parameter.2036118857.value", "example"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "parameter.484080973.name", "enable_user_activity_logging"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "parameter.484080973.value", "true"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftParameterGroup_withoutParameters(t *testing.T) {
	var v redshift.ClusterParameterGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftParameterGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSRedshiftParameterGroupOnlyConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftParameterGroupExists("aws_redshift_parameter_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "name", "parameter-group-test-terraform"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "family", "redshift-1.0"),
					resource.TestCheckResourceAttr(
						"aws_redshift_parameter_group.bar", "description", "Test parameter group for terraform"),
				),
			},
		},
	})
}

func TestResourceAWSRedshiftParameterGroupNameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "tEsting123",
			ErrCount: 1,
		},
		{
			Value:    "testing123!",
			ErrCount: 1,
		},
		{
			Value:    "1testing123",
			ErrCount: 1,
		},
		{
			Value:    "testing--123",
			ErrCount: 1,
		},
		{
			Value:    "testing123-",
			ErrCount: 1,
		},
		{
			Value:    randomString(256),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftParamGroupName(tc.Value, "aws_redshift_parameter_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Parameter Group Name to trigger a validation error")
		}
	}
}

func testAccCheckAWSRedshiftParameterGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).redshiftconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_redshift_parameter_group" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeClusterParameterGroups(
			&redshift.DescribeClusterParameterGroupsInput{
				ParameterGroupName: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.ParameterGroups) != 0 &&
				*resp.ParameterGroups[0].ParameterGroupName == rs.Primary.ID {
				return fmt.Errorf("Redshift Parameter Group still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "ClusterParameterGroupNotFound" {
			return err
		}
	}

	return nil
}

func testAccCheckAWSRedshiftParameterGroupExists(n string, v *redshift.ClusterParameterGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Redshift Parameter Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).redshiftconn

		opts := redshift.DescribeClusterParameterGroupsInput{
			ParameterGroupName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeClusterParameterGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.ParameterGroups) != 1 ||
			*resp.ParameterGroups[0].ParameterGroupName != rs.Primary.ID {
			return fmt.Errorf("Redshift Parameter Group not found")
		}

		*v = *resp.ParameterGroups[0]

		return nil
	}
}

const testAccAWSRedshiftParameterGroupOnlyConfig = `
resource "aws_redshift_parameter_group" "bar" {
	name = "parameter-group-test-terraform"
	family = "redshift-1.0"
	description = "Test parameter group for terraform"
}`

const testAccAWSRedshiftParameterGroupConfig = `
resource "aws_redshift_parameter_group" "bar" {
	name = "parameter-group-test-terraform"
	family = "redshift-1.0"
	description = "Test parameter group for terraform"
	parameter {
	  name = "require_ssl"
	  value = "true"
	}
	parameter {
	  name = "query_group"
	  value = "example"
	}
	parameter{
	  name = "enable_user_activity_logging"
	  value = "true"
	}
}
`
