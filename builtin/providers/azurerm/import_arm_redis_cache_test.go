package azurerm

import (
	"testing"

	"fmt"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureRMRedisCache_importBasic(t *testing.T) {
	resourceName := "azurerm_redis_cache.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"redis_configuration"},
			},
		},
	})
}

func TestAccAzureRMRedisCache_importStandard(t *testing.T) {
	resourceName := "azurerm_redis_cache.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_standard, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"redis_configuration"},
			},
		},
	})
}

func TestAccAzureRMRedisCache_importPremium(t *testing.T) {
	resourceName := "azurerm_redis_cache.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_premium, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"redis_configuration"},
			},
		},
	})
}

func TestAccAzureRMRedisCache_importPremiumSharded(t *testing.T) {
	resourceName := "azurerm_redis_cache.test"

	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_premiumSharded, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},

			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"redis_configuration"},
			},
		},
	})
}
