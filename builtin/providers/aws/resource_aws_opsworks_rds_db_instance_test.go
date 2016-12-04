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
				Config: testAccAwsOpsworksRdsDbInstance(sName, "foo", "barbarbarbar"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "foo",
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsOpsworksRdsDbInstance(sName, "bar", "barbarbarbar"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "bar",
					),
				),
			},
			resource.TestStep{
				Config: testAccAwsOpsworksRdsDbInstance(sName, "bar", "foofoofoofoofoo"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"aws_opsworks_rds_db_instance.tf-acc-opsworks-db", "db_user", "bar",
					),
				),
			},
		},
	})
}

func testAccAwsOpsworksRdsDbInstance(name, userName, password string) string {
	return fmt.Sprintf(`
resource "aws_opsworks_rds_db_instance" "tf-acc-opsworks-db" {
  stack_id = "${aws_opsworks_stack.tf-acc.id}"

  rds_db_instance_arn = "${aws_db_instance.bar.arn}"
  db_user = "%s"
  db_password = "%s"
}

%s

%s
`, userName, password, testAccAwsOpsworksStackConfigVpcCreate(name), testAccAWSDBInstanceConfig)
}
