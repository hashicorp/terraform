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

func TestAccAwsDmsEndpointBasic(t *testing.T) {
	resourceName := "aws_dms_endpoint.dms_endpoint"
	randId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsEndpointDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsEndpointConfig(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsEndpointExists(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "endpoint_arn"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
			{
				Config: dmsEndpointConfigUpdate(randId),
				Check: resource.ComposeTestCheckFunc(
					checkDmsEndpointExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "database_name", "tf-test-dms-db-updated"),
					resource.TestCheckResourceAttr(resourceName, "extra_connection_attributes", "extra"),
					resource.TestCheckResourceAttr(resourceName, "password", "tftestupdate"),
					resource.TestCheckResourceAttr(resourceName, "port", "3303"),
					resource.TestCheckResourceAttr(resourceName, "ssl_mode", "none"),
					resource.TestCheckResourceAttr(resourceName, "server_name", "tftestupdate"),
					resource.TestCheckResourceAttr(resourceName, "username", "tftestupdate"),
				),
			},
		},
	})
}

func dmsEndpointDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_dms_endpoint" {
			continue
		}

		err := checkDmsEndpointExists(rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("Found an endpoint that was not destroyed: %s", rs.Primary.ID)
		}
	}

	return nil
}

func checkDmsEndpointExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).dmsconn
		resp, err := conn.DescribeEndpoints(&dms.DescribeEndpointsInput{
			Filters: []*dms.Filter{
				{
					Name:   aws.String("endpoint-id"),
					Values: []*string{aws.String(rs.Primary.ID)},
				},
			},
		})

		if err != nil {
			return fmt.Errorf("DMS endpoint error: %v", err)
		}

		if resp.Endpoints == nil {
			return fmt.Errorf("DMS endpoint not found")
		}

		return nil
	}
}

func dmsEndpointConfig(randId string) string {
	return fmt.Sprintf(`
resource "aws_dms_endpoint" "dms_endpoint" {
	database_name = "tf-test-dms-db"
	endpoint_id = "tf-test-dms-endpoint-%[1]s"
	endpoint_type = "source"
	engine_name = "aurora"
	extra_connection_attributes = ""
	password = "tftest"
	port = 3306
	server_name = "tftest"
	ssl_mode = "none"
	tags {
		Name = "tf-test-dms-endpoint-%[1]s"
		Update = "to-update"
		Remove = "to-remove"
	}
	username = "tftest"
}
`, randId)
}

func dmsEndpointConfigUpdate(randId string) string {
	return fmt.Sprintf(`
resource "aws_dms_endpoint" "dms_endpoint" {
	database_name = "tf-test-dms-db-updated"
	endpoint_id = "tf-test-dms-endpoint-%[1]s"
	endpoint_type = "source"
	engine_name = "aurora"
	extra_connection_attributes = "extra"
	password = "tftestupdate"
	port = 3303
	server_name = "tftestupdate"
	ssl_mode = "none"
	tags {
		Name = "tf-test-dms-endpoint-%[1]s"
		Update = "updated"
		Add = "added"
	}
	username = "tftestupdate"
}
`, randId)
}
