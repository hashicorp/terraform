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
				),
			},
		},
	})
}

func testAccCheckAwsSsmParameterDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "aws_ssm_parameter" "test" {
	name = "%s"
}
`, name)
}
