package aws

import (
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccAwsDmsReplicationSubnetGroupImport(t *testing.T) {
	resourceName := "aws_dms_replication_subnet_group.dms_replication_subnet_group"
	randId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsReplicationSubnetGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsReplicationSubnetGroupConfig(randId),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
