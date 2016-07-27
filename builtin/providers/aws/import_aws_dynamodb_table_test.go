package aws

import (
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAWSDynamoDbTable_importBasic(t *testing.T) {
	resourceName := "aws_dynamodb_table.basic-dynamodb-table"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSDynamoDbTableDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSDynamoDbConfigInitialState(),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
