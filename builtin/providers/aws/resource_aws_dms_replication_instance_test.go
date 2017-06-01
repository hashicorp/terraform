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

func TestAccAwsDmsReplicationInstanceBasic(t *testing.T) {
	resourceName := "aws_dms_replication_instance.dms_replication_instance"
	randId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsReplicationInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsReplicationInstanceConfig(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsReplicationInstanceExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "replication_instance_arn"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: dmsReplicationInstanceConfigUpdate(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsReplicationInstanceExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "apply_immediately"),
					resource.TestCheckResourceAttr(resourceName, "auto_minor_version_upgrade", "false"),
					resource.TestCheckResourceAttr(resourceName, "preferred_maintenance_window", "mon:00:30-mon:02:30"),
				),
			},
		},
	})
}

func checkDmsReplicationInstanceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}
		conn := testAccProvider.Meta().(*AWSClient).dmsconn
		resp, err := conn.DescribeReplicationInstances(&dms.DescribeReplicationInstancesInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("replication-instance-id"),
					Values: []*string{aws.String(rs.Primary.ID)},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("DMS replication instance error: %v", err)
		}
		if resp.ReplicationInstances == nil {
			return fmt.Errorf("DMS replication instance not found")
		}

		return nil
	}
}

func dmsReplicationInstanceDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dms_replication_instance" {
			continue
		}

		err := checkDmsReplicationInstanceExists(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Found replication instance that was not destroyed: %s", rs.Primary.ID)
		}
	}

	return nil
}

func dmsReplicationInstanceConfig(randId string) string {
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

resource "aws_dms_replication_subnet_group" "dms_replication_subnet_group" {
	replication_subnet_group_id = "tf-test-dms-replication-subnet-group-%[1]s"
	replication_subnet_group_description = "terraform test for replication subnet group"
	subnet_ids = ["${aws_subnet.dms_subnet_1.id}", "${aws_subnet.dms_subnet_2.id}"]
}

resource "aws_dms_replication_instance" "dms_replication_instance" {
	allocated_storage = 5
	auto_minor_version_upgrade = true
	replication_instance_class = "dms.t2.micro"
	replication_instance_id = "tf-test-dms-replication-instance-%[1]s"
	preferred_maintenance_window = "sun:00:30-sun:02:30"
	publicly_accessible = false
	replication_subnet_group_id = "${aws_dms_replication_subnet_group.dms_replication_subnet_group.replication_subnet_group_id}"
	tags {
		Name = "tf-test-dms-replication-instance-%[1]s"
		Update = "to-update"
		Remove = "to-remove"
	}

	timeouts {
		create = "40m"
	}
}
`, randId)
}

func dmsReplicationInstanceConfigUpdate(randId string) string {
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

resource "aws_dms_replication_subnet_group" "dms_replication_subnet_group" {
	replication_subnet_group_id = "tf-test-dms-replication-subnet-group-%[1]s"
	replication_subnet_group_description = "terraform test for replication subnet group"
	subnet_ids = ["${aws_subnet.dms_subnet_1.id}", "${aws_subnet.dms_subnet_2.id}"]
}

resource "aws_dms_replication_instance" "dms_replication_instance" {
	allocated_storage = 5
	apply_immediately = true
	auto_minor_version_upgrade = false
	replication_instance_class = "dms.t2.micro"
	replication_instance_id = "tf-test-dms-replication-instance-%[1]s"
	preferred_maintenance_window = "mon:00:30-mon:02:30"
	publicly_accessible = false
	replication_subnet_group_id = "${aws_dms_replication_subnet_group.dms_replication_subnet_group.replication_subnet_group_id}"
	tags {
		Name = "tf-test-dms-replication-instance-%[1]s"
		Update = "updated"
		Add = "added"
	}
}
`, randId)
}
