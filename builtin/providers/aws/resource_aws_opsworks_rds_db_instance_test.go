package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSOpsworksRdsDbInstance(t *testing.T) {
	sName := fmt.Sprintf("test-db-instance-%d", acctest.RandInt())
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAwsOpsworksRdsDbInstance(sName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "foo",
					),
				),
			},
		},
	})
}

func testAccAwsOpsworksRdsDbInstance(name string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_rds_db_instance" "tf-acc-opsworks-db" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  rds_db_instance_arn = "${aws_db_instance.bar.arn}"
  db_user = "${aws_db_instance.bar.username}"
  db_password = "${aws_db_instance.bar.password}"
}

%s

%s
`, testAccAwsOpsworksStackConfigNoVpcCreate(name), testAccAWSDBInstanceConfig)
}
