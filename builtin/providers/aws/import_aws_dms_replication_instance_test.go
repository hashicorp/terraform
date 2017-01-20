package aws

import (
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccAwsDmsReplicationInstanceImport(t *testing.T) {
	resourceName := "aws_dms_replication_instance.dms_replication_instance"
	randId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsReplicationInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsReplicationInstanceConfig(randId),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
