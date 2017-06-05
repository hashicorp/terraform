package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	dms "github.com/aws/aws-sdk-go/service/databasemigrationservice"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAwsDmsReplicationSubnetGroupBasic(t *testing.T) {
	resourceName := "aws_dms_replication_subnet_group.dms_replication_subnet_group"
	randId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsReplicationSubnetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsReplicationSubnetGroupConfig(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsReplicationSubnetGroupExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "vpc_id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: dmsReplicationSubnetGroupConfigUpdate(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsReplicationSubnetGroupExists(resourceName),
				),
			},
		},
	})
}

func checkDmsReplicationSubnetGroupExists(n string) resource.TestCheckFunc {
	providers := []*schema.Provider{testAccProvider}
	return checkDmsReplicationSubnetGroupExistsWithProviders(n, &providers)
}

func checkDmsReplicationSubnetGroupExistsWithProviders(n string, providers *[]*schema.Provider) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		for _, provider := range *providers {
			// Ignore if Meta is empty, this can happen for validation providers
			if provider.Meta() == nil {
				continue
			}

			conn := provider.Meta().(*AWSClient).dmsconn
			_, err := conn.DescribeReplicationSubnetGroups(&dms.DescribeReplicationSubnetGroupsInput{
				Filters: []*dms.Filter{
					{
						Name:   aws.String("replication-subnet-group-id"),
						Values: []*string{aws.String(rs.Primary.ID)},
					},
				},
			})

			if err != nil {
				return fmt.Errorf("DMS replication subnet group error: %v", err)
			}
			return nil
		}

		return fmt.Errorf("DMS replication subnet group not found")
	}
}

func dmsReplicationSubnetGroupDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dms_replication_subnet_group" {
			continue
		}

		err := checkDmsReplicationSubnetGroupExists(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Found replication subnet group that was not destroyed: %s", rs.Primary.ID)
		}
	}

	return nil
}

func dmsReplicationSubnetGroupConfig(randId string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "dms_vpc" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "tf-test-dms-vpc-%[1]s"
	}
}

resource "aws_subnet" "dms_subnet_1" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_subnet" "dms_subnet_2" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_subnet" "dms_subnet_3" {
	cidr_block = "10.1.3.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_dms_replication_subnet_group" "dms_replication_subnet_group" {
	replication_subnet_group_id = "tf-test-dms-replication-subnet-group-%[1]s"
	replication_subnet_group_description = "terraform test for replication subnet group"
	subnet_ids = ["${aws_subnet.dms_subnet_1.id}", "${aws_subnet.dms_subnet_2.id}"]
	tags {
		Name = "tf-test-dms-replication-subnet-group-%[1]s"
		Update = "to-update"
		Remove = "to-remove"
	}
}
`, randId)
}

func dmsReplicationSubnetGroupConfigUpdate(randId string) string {
	return fmt.Sprintf(`
resource "aws_vpc" "dms_vpc" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "tf-test-dms-vpc-%[1]s"
	}
}

resource "aws_subnet" "dms_subnet_1" {
	cidr_block = "10.1.1.0/24"
	availability_zone = "us-west-2a"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_subnet" "dms_subnet_2" {
	cidr_block = "10.1.2.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_subnet" "dms_subnet_3" {
	cidr_block = "10.1.3.0/24"
	availability_zone = "us-west-2b"
	vpc_id = "${aws_vpc.dms_vpc.id}"
	tags {
		Name = "tf-test-dms-subnet-%[1]s"
	}
	depends_on = ["aws_vpc.dms_vpc"]
}

resource "aws_dms_replication_subnet_group" "dms_replication_subnet_group" {
	replication_subnet_group_id = "tf-test-dms-replication-subnet-group-%[1]s"
	replication_subnet_group_description = "terraform test for replication subnet group"
	subnet_ids = ["${aws_subnet.dms_subnet_1.id}", "${aws_subnet.dms_subnet_3.id}"]
	tags {
		Name = "tf-test-dms-replication-subnet-group-%[1]s"
		Update = "updated"
		Add = "added"
	}
}
`, randId)
}
