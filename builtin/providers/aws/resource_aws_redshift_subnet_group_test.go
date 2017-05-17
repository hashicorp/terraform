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

func TestAccAWSRedshiftSubnetGroup_basic(t *testing.T) {
	var v redshift.ClusterSubnetGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRedshiftSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRedshiftSubnetGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRedshiftSubnetGroupExists("aws_redshift_subnet_group.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "subnet_ids.#", "2"),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "description", "foo description"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftSubnetGroup_updateDescription(t *testing.T) {
	var v redshift.ClusterSubnetGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRedshiftSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRedshiftSubnetGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRedshiftSubnetGroupExists("aws_redshift_subnet_group.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "description", "foo description"),
				),
			},

			resource.TestStep{
				Config: testAccRedshiftSubnetGroup_updateDescription(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRedshiftSubnetGroupExists("aws_redshift_subnet_group.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "description", "foo description updated"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftSubnetGroup_updateSubnetIds(t *testing.T) {
	var v redshift.ClusterSubnetGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRedshiftSubnetGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccRedshiftSubnetGroupConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRedshiftSubnetGroupExists("aws_redshift_subnet_group.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "subnet_ids.#", "2"),
				),
			},

			resource.TestStep{
				Config: testAccRedshiftSubnetGroupConfig_updateSubnetIds(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRedshiftSubnetGroupExists("aws_redshift_subnet_group.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "subnet_ids.#", "3"),
				),
			},
		},
	})
}

func TestAccAWSRedshiftSubnetGroup_tags(t *testing.T) {
	var v redshift.ClusterSubnetGroup
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckRedshiftSubnetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccRedshiftSubnetGroupConfigWithTags(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRedshiftSubnetGroupExists("aws_redshift_subnet_group.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "tags.%", "1"),
					resource.TestCheckResourceAttr("aws_redshift_subnet_group.foo", "tags.Name", "tf-redshift-subnetgroup"),
				),
			},
			{
				Config: testAccRedshiftSubnetGroupConfigWithTagsUpdated(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRedshiftSubnetGroupExists("aws_redshift_subnet_group.foo", &v),
					resource.TestCheckResourceAttr(
						"aws_redshift_subnet_group.foo", "tags.%", "3"),
					resource.TestCheckResourceAttr("aws_redshift_subnet_group.foo", "tags.environment", "production"),
					resource.TestCheckResourceAttr("aws_redshift_subnet_group.foo", "tags.Name", "tf-redshift-subnetgroup"),
					resource.TestCheckResourceAttr("aws_redshift_subnet_group.foo", "tags.foo", "bar"),
				),
			},
		},
	})
}

func TestResourceAWSRedshiftSubnetGroupNameValidation(t *testing.T) {
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
			Value:    "testing_123",
			ErrCount: 1,
		},
		{
			Value:    "testing.123",
			ErrCount: 1,
		},
		{
			Value:    randomString(256),
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedshiftSubnetGroupName(tc.Value, "aws_redshift_subnet_group_name")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Redshift Subnet Group Name to trigger a validation error")
		}
	}
}

func testAccCheckRedshiftSubnetGroupDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).redshiftconn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_redshift_subnet_group" {
			continue
		}

		resp, err := conn.DescribeClusterSubnetGroups(
			&redshift.DescribeClusterSubnetGroupsInput{
				ClusterSubnetGroupName: aws.String(rs.Primary.ID)})
		if err == nil {
			if len(resp.ClusterSubnetGroups) > 0 {
				return fmt.Errorf("still exist.")
			}

			return nil
		}

		redshiftErr, ok := err.(awserr.Error)
		if !ok {
			return err
		}
		if redshiftErr.Code() != "ClusterSubnetGroupNotFoundFault" {
			return err
		}
	}

	return nil
}

func testAccCheckRedshiftSubnetGroupExists(n string, v *redshift.ClusterSubnetGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).redshiftconn
		resp, err := conn.DescribeClusterSubnetGroups(
			&redshift.DescribeClusterSubnetGroupsInput{ClusterSubnetGroupName: aws.String(rs.Primary.ID)})
		if err != nil {
			return err
		}
		if len(resp.ClusterSubnetGroups) == 0 {
			return fmt.Errorf("ClusterSubnetGroup not found")
		}

		*v = *resp.ClusterSubnetGroups[0]

		return nil
	}
}

func testAccRedshiftSubnetGroupConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccRedshiftSubnetGroupConfig"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-1"
	}
}

resource "aws_subnet" "bar" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-2"
	}
}

resource "aws_redshift_subnet_group" "foo" {
	name = "foo-%d"
	description = "foo description"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}
	`, rInt)
}

func testAccRedshiftSubnetGroup_updateDescription(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccRedshiftSubnetGroup_updateDescription"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-1"
	}
}

resource "aws_subnet" "bar" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-2"
	}
}

resource "aws_redshift_subnet_group" "foo" {
	name = "foo-%d"
	description = "foo description updated"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
}
`, rInt)
}

func testAccRedshiftSubnetGroupConfigWithTags(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccRedshiftSubnetGroupConfigWithTags"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-1"
	}
}

resource "aws_subnet" "bar" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-2"
	}
}

resource "aws_redshift_subnet_group" "foo" {
	name = "foo-%d"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
	tags {
		Name = "tf-redshift-subnetgroup"
	}
}
`, rInt)
}

func testAccRedshiftSubnetGroupConfigWithTagsUpdated(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccRedshiftSubnetGroupConfigWithTags"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-1"
	}
}

resource "aws_subnet" "bar" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-2"
	}
}

resource "aws_redshift_subnet_group" "foo" {
	name = "foo-%d"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}"]
	tags {
		Name = "tf-redshift-subnetgroup"
		environment = "production"
		foo = "bar"
	}
}
`, rInt)
}

func testAccRedshiftSubnetGroupConfig_updateSubnetIds(rInt int) string {
	return fmt.Sprintf(`
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccRedshiftSubnetGroupConfig_updateSubnetIds"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-1"
	}
}

resource "aws_subnet" "bar" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-2"
	}
}

resource "aws_subnet" "foobar" {
	cidr_block = "10.1.3.0/24"
	availability_zone = "us-west-2c"
	vpc_id = "${aws_vpc.foo.id}"
	tags {
		Name = "tf-dbsubnet-test-3"
	}
}

resource "aws_redshift_subnet_group" "foo" {
	name = "foo-%d"
	subnet_ids = ["${aws_subnet.foo.id}", "${aws_subnet.bar.id}", "${aws_subnet.foobar.id}"]
}
`, rInt)
}
