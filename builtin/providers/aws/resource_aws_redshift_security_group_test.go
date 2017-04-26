package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSRedshiftSecurityGroup_ingressCidr(t *testing.T) {
	var v redshift.ClusterSecurityGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressCidr(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "name", fmt.Sprintf("redshift-sg-terraform-%d", rInt)),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.2735652665.cidr", "10.0.0.1/24"),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftSecurityGroup_updateIngressCidr(t *testing.T) {
	var v redshift.ClusterSecurityGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressCidr(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "1"),
				),
			},

			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressCidrAdd(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "3"),
				),
			},

			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressCidrReduce(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "2"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftSecurityGroup_ingressSecurityGroup(t *testing.T) {
	var v redshift.ClusterSecurityGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressSgId(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "name", fmt.Sprintf("redshift-sg-terraform-%d", rInt)),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "description", "this is a description"),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "1"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftSecurityGroup_updateIngressSecurityGroup(t *testing.T) {
	var v redshift.ClusterSecurityGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSRedshiftSecurityGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressSgId(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "1"),
				),
			},

			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressSgIdAdd(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "3"),
				),
			},

			{
				Config: testAccAWSRedshiftSecurityGroupConfig_ingressSgIdReduce(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSRedshiftSecurityGroupExists("aws_redshift_security_group.bar", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_security_group.bar", "ingress.#", "2"),
				),
			},
		},
	})
}

func testAccCheckAWSRedshiftSecurityGroupExists(n string, v *redshift.ClusterSecurityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Redshift Security Group ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).redshiftconn

		opts := redshift.DescribeClusterSecurityGroupsInput{
			ClusterSecurityGroupName: aws.String(rs.Primary.ID),
		}

		resp, err := conn.DescribeClusterSecurityGroups(&opts)

		if err != nil {
			return err
		}

		if len(resp.ClusterSecurityGroups) != 1 ||
			*resp.ClusterSecurityGroups[0].ClusterSecurityGroupName != rs.Primary.ID {
			return fmt.Errorf("Redshift Security Group not found")
		}

		*v = *resp.ClusterSecurityGroups[0]

		return nil
	}
}

func testAccCheckAWSRedshiftSecurityGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).redshiftconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_redshift_security_group" {
			continue
		}

		// Try to find the Group
		resp, err := conn.DescribeClusterSecurityGroups(
			&redshift.DescribeClusterSecurityGroupsInput{
				ClusterSecurityGroupName: aws.String(rs.Primary.ID),
			})

		if err == nil {
			if len(resp.ClusterSecurityGroups) != 0 &&
				*resp.ClusterSecurityGroups[0].ClusterSecurityGroupName == rs.Primary.ID {
				return fmt.Errorf("Redshift Security Group still exists")
			}
		}

		// Verify the error
		newerr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if newerr.Code() != "ClusterSecurityGroupNotFound" {
			return err
		}
	}

	return nil
}

func TestResourceAWSRedshiftSecurityGroupNameValidation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "default",
			ErrCount: 1,
		},
		{
			Value:    "testing123%%",
			ErrCount: 1,
		},
		{
			Value:    "TestingSG",
			ErrCount: 1,
		},
		{
			Value:    randomString(256),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftSecurityGroupName(tc.Value, "aws_redshift_security_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Security Group Name to trigger a validation error")
		}
	}
}

func testAccAWSRedshiftSecurityGroupConfig_ingressCidr(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
			region = "us-east-1"
	}

	resource "aws_redshift_security_group" "bar" {
			name = "redshift-sg-terraform-%d"

			ingress {
					cidr = "10.0.0.1/24"
			}
	}`, rInt)
}

func testAccAWSRedshiftSecurityGroupConfig_ingressCidrAdd(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
			region = "us-east-1"
	}

	resource "aws_redshift_security_group" "bar" {
			name = "redshift-sg-terraform-%d"
			description = "this is a description"

			ingress {
					cidr = "10.0.0.1/24"
			}

			ingress {
					cidr = "10.0.10.1/24"
			}

			ingress {
					cidr = "10.0.20.1/24"
			}
	}`, rInt)
}

func testAccAWSRedshiftSecurityGroupConfig_ingressCidrReduce(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
			region = "us-east-1"
	}

	resource "aws_redshift_security_group" "bar" {
			name = "redshift-sg-terraform-%d"
			description = "this is a description"

			ingress {
					cidr = "10.0.0.1/24"
			}

			ingress {
					cidr = "10.0.10.1/24"
			}
	}`, rInt)
}

func testAccAWSRedshiftSecurityGroupConfig_ingressSgId(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
			region = "us-east-1"
	}

	resource "aws_security_group" "redshift" {
		name = "terraform_redshift_test_%d"
		description = "Used in the redshift acceptance tests"

		ingress {
			protocol = "tcp"
			from_port = 22
			to_port = 22
			cidr_blocks = ["10.0.0.0/8"]
		}
	}

	resource "aws_redshift_security_group" "bar" {
			name = "redshift-sg-terraform-%d"
			description = "this is a description"

			ingress {
					security_group_name = "${aws_security_group.redshift.name}"
					security_group_owner_id = "${aws_security_group.redshift.owner_id}"
			}
	}`, rInt, rInt)
}

func testAccAWSRedshiftSecurityGroupConfig_ingressSgIdAdd(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
			region = "us-east-1"
	}

	resource "aws_security_group" "redshift" {
		name = "terraform_redshift_test_%d"
		description = "Used in the redshift acceptance tests"

		ingress {
			protocol = "tcp"
			from_port = 22
			to_port = 22
			cidr_blocks = ["10.0.0.0/16"]
		}
	}

	resource "aws_security_group" "redshift2" {
		name = "terraform_redshift_test_2_%d"
		description = "Used in the redshift acceptance tests #2"

		ingress {
			protocol = "tcp"
			from_port = 22
			to_port = 22
			cidr_blocks = ["10.1.0.0/16"]
		}
	}

	resource "aws_security_group" "redshift3" {
		name = "terraform_redshift_test_3_%d"
		description = "Used in the redshift acceptance tests #3"

		ingress {
			protocol = "tcp"
			from_port = 22
			to_port = 22
			cidr_blocks = ["10.2.0.0/16"]
		}
	}

	resource "aws_redshift_security_group" "bar" {
			name = "redshift-sg-terraform-%d"
			description = "this is a description"

			ingress {
					security_group_name = "${aws_security_group.redshift.name}"
					security_group_owner_id = "${aws_security_group.redshift.owner_id}"
			}

			ingress {
					security_group_name = "${aws_security_group.redshift2.name}"
					security_group_owner_id = "${aws_security_group.redshift.owner_id}"
			}

			ingress {
					security_group_name = "${aws_security_group.redshift3.name}"
					security_group_owner_id = "${aws_security_group.redshift.owner_id}"
			}
	}`, rInt, rInt, rInt, rInt)
}

func testAccAWSRedshiftSecurityGroupConfig_ingressSgIdReduce(rInt int) string {
	return fmt.Sprintf(`
	provider "aws" {
			region = "us-east-1"
	}

	resource "aws_security_group" "redshift" {
		name = "terraform_redshift_test_%d"
		description = "Used in the redshift acceptance tests"

		ingress {
			protocol = "tcp"
			from_port = 22
			to_port = 22
			cidr_blocks = ["10.0.0.0/16"]
		}
	}

	resource "aws_security_group" "redshift2" {
		name = "terraform_redshift_test_2_%d"
		description = "Used in the redshift acceptance tests #2"

		ingress {
			protocol = "tcp"
			from_port = 22
			to_port = 22
			cidr_blocks = ["10.1.0.0/16"]
		}
	}

	resource "aws_redshift_security_group" "bar" {
			name = "redshift-sg-terraform-%d"
			description = "this is a description"

			ingress {
					security_group_name = "${aws_security_group.redshift.name}"
					security_group_owner_id = "${aws_security_group.redshift.owner_id}"
			}

			ingress {
					security_group_name = "${aws_security_group.redshift2.name}"
					security_group_owner_id = "${aws_security_group.redshift.owner_id}"
			}
	}`, rInt, rInt, rInt)
}
