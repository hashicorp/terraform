package google

import (
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccGoogleSqlUser_importBasic(t *testing.T) {
	resourceName := "google_sql_user.user"
	user := acctest.RandString(10)
	instance := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccGoogleSqlUserDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testGoogleSqlUser_basic(instance, user),
			},

			resource.TestStep{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}
