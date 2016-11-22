package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSDBSnapshot_basic(t *testing.T) {
	var v rds.DBSnapshot
	rInt := acctest.RandInt()
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccAwsDbSnapshotConfig(rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDbSnapshotExists("aws_db_snapshot.test", &v),
				),
			},
		},
	})
}

func testAccCheckDbSnapshotExists(n string, v *rds.DBSnapshot) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).rdsconn

		request := &rds.DescribeDBSnapshotsInput{
			DBSnapshotIdentifier: aws.String(rs.Primary.ID),
		}

		response, err := conn.DescribeDBSnapshots(request)
		if err == nil {
			if response.DBSnapshots != nil && len(response.DBSnapshots) > 0 {
				*v = *response.DBSnapshots[0]
				return nil
			}
		}
		return fmt.Errorf("Error finding RDS DB Snapshot %s", rs.Primary.ID)
	}
}

func testAccAwsDbSnapshotConfig(rInt int) string {
	return fmt.Sprintf(`
resource "aws_db_instance" "bar" {
	allocated_storage = 10
	engine = "MySQL"
	engine_version = "5.6.21"
	instance_class = "db.t1.micro"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"

	maintenance_window = "Fri:09:00-Fri:09:30"

	backup_retention_period = 0

	parameter_group_name = "default.mysql5.6"

	skip_final_snapshot = true
}

resource "aws_db_snapshot" "test" {
	db_instance_identifier = "${aws_db_instance.bar.id}"
	db_snapshot_identifier = "testsnapshot%d"
}`, rInt)
}
