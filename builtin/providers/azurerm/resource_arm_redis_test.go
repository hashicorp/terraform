package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMRedisFamily_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "C",
			ErrCount: 0,
		},
		{
			Value:    "P",
			ErrCount: 0,
		},
		{
			Value:    "c",
			ErrCount: 0,
		},
		{
			Value:    "p",
			ErrCount: 0,
		},
		{
			Value:    "a",
			ErrCount: 1,
		},
		{
			Value:    "b",
			ErrCount: 1,
		},
		{
			Value:    "D",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedisFamily(tc.Value, "azurerm_redis")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Redis Family to trigger a validation error")
		}
	}
}

func TestAccAzureRMRedisSku_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{
			Value:    "Basic",
			ErrCount: 0,
		},
		{
			Value:    "Standard",
			ErrCount: 0,
		},
		{
			Value:    "Premium",
			ErrCount: 0,
		},
		{
			Value:    "Random",
			ErrCount: 1,
		},
	}

	for _, tc := range cases {
		_, errors := validateRedisSku(tc.Value, "azurerm_redis")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Redis Sku to trigger a validation error")
		}
	}
}

func TestAccAzureRMRedis_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedis_basic, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisExists("azurerm_redis.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRedis_standard(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedis_standard, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisExists("azurerm_redis.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRedis_premium(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedis_premium, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisExists("azurerm_redis.test"),
				),
			},
		},
	})
}

func testCheckAzureRMRedisExists(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Ensure we have enough information in state to look up in API
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		redisName := rs.Primary.Attributes["name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for Redis Instance: %s", redisName)
		}

		conn := testAccProvider.Meta().(*ArmClient).redisClient

		resp, err := conn.Get(resourceGroup, redisName)
		if err != nil {
			return fmt.Errorf("Bad: Get on redisClient: %s", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("Bad: Redis Instance %q (resource group: %q) does not exist", redisName, resourceGroup)
		}

		return nil
	}
}

func testCheckAzureRMRedisDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).redisClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_redis" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Redis Instance still exists:\n%#v", resp.Properties)
		}
	}

	return nil
}

var testAccAzureRMRedis_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_redis" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    redis_version       = "3.0"
    capacity            = 1
    family              = "C"
    sku_name            = "Basic"
    enable_non_ssl_port = false
}
`

var testAccAzureRMRedis_standard = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_redis" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    redis_version       = "3.0"
    capacity            = 1
    family              = "C"
    sku_name            = "Standard"
    enable_non_ssl_port = false
}
`

var testAccAzureRMRedis_premium = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_redis" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    redis_version       = "3.0"
    capacity            = 1
    family              = "C"
    sku_name            = "Premium"
    enable_non_ssl_port = false
    shard_count         = 3
    redis_configuration {
      "maxclients"         = "256",
      "maxmemory-reserved" = "2",
      "maxmemory-delta"    = "2"
      "maxmemory-policy"   = "allkeys-lru"
    }
}
`
