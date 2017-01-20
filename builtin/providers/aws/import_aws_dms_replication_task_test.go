package aws

import (
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"testing"
)

func TestAccAwsDmsReplicationTaskImport(t *testing.T) {
	resourceName := "aws_dms_replication_task.dms_replication_task"
	randId := acctest.RandString(8)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: dmsReplicationTaskDestroy,
		Steps: []resource.TestStep{
			{
				Config: dmsReplicationTaskConfig(randId),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
