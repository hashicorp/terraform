package aws

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAwsSsmParameterDataSource_basic(t *testing.T) {
	name := "test.parameter"
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsSsmParameterDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.aws_ssm_parameter.test", "name", name),
					resource.TestCheckResourceAttr("data.aws_ssm_parameter.test", "type", "String"),
					resource.TestCheckResourceAttr("data.aws_ssm_parameter.test", "value", "TestValue"),
				),
			},
		},
	})
}

func testAccCheckAwsSsmParameterDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "aws_ssm_parameter" "test" {
	name = "%s"
	type = "String"
	value = "TestValue"
}

data "aws_ssm_parameter" "test" {
	name = "${aws_ssm_parameter.test.name}"
}
`, name)
}
