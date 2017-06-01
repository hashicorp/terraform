package azurerm

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureRMRedisCacheFamily_validation(t *testing.T) {
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
		_, errors := validateRedisFamily(tc.Value, "azurerm_redis_cache")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Redis Cache Family to trigger a validation error")
		}
	}
}

func TestAccAzureRMRedisCacheMaxMemoryPolicy_validation(t *testing.T) {
	cases := []struct {
		Value    string
		ErrCount int
	}{
		{Value: "noeviction", ErrCount: 0},
		{Value: "allkeys-lru", ErrCount: 0},
		{Value: "volatile-lru", ErrCount: 0},
		{Value: "allkeys-random", ErrCount: 0},
		{Value: "volatile-random", ErrCount: 0},
		{Value: "volatile-ttl", ErrCount: 0},
		{Value: "something-else", ErrCount: 1},
	}

	for _, tc := range cases {
		_, errors := validateRedisMaxMemoryPolicy(tc.Value, "azurerm_redis_cache")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Redis Cache Max Memory Policy to trigger a validation error")
		}
	}
}

func TestAccAzureRMRedisCacheSku_validation(t *testing.T) {
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
		_, errors := validateRedisSku(tc.Value, "azurerm_redis_cache")

		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected the Azure RM Redis Cache Sku to trigger a validation error")
		}
	}
}

func TestAccAzureRMRedisCache_basic(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_basic, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisCacheExists("azurerm_redis_cache.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRedisCache_standard(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_standard, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisCacheExists("azurerm_redis_cache.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRedisCache_premium(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_premium, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisCacheExists("azurerm_redis_cache.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRedisCache_premiumSharded(t *testing.T) {
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccAzureRMRedisCache_premiumSharded, ri, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisCacheExists("azurerm_redis_cache.test"),
				),
			},
		},
	})
}

func TestAccAzureRMRedisCache_NonStandardCasing(t *testing.T) {
	ri := acctest.RandInt()
	config := testAccAzureRMRedisCacheNonStandardCasing(ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureRMRedisCacheDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureRMRedisCacheExists("azurerm_redis_cache.test"),
				),
			},

			resource.TestStep{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func testCheckAzureRMRedisCacheExists(name string) resource.TestCheckFunc {
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

func testCheckAzureRMRedisCacheDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*ArmClient).redisClient

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurerm_redis_cache" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		resourceGroup := rs.Primary.Attributes["resource_group_name"]

		resp, err := conn.Get(resourceGroup, name)

		if err != nil {
			return nil
		}

		if resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Redis Instance still exists:\n%#v", resp)
		}
	}

	return nil
}

var testAccAzureRMRedisCache_basic = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_redis_cache" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    capacity            = 1
    family              = "C"
    sku_name            = "Basic"
    enable_non_ssl_port = false

    redis_configuration {
      maxclients = "256"
    }
}
`

var testAccAzureRMRedisCache_standard = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_redis_cache" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    capacity            = 1
    family              = "C"
    sku_name            = "Standard"
    enable_non_ssl_port = false
    redis_configuration {
      maxclients = "256"
    }

    tags {
    	environment = "production"
    }
}
`

var testAccAzureRMRedisCache_premium = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_redis_cache" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    capacity            = 1
    family              = "P"
    sku_name            = "Premium"
    enable_non_ssl_port = false
    redis_configuration {
      maxclients         = "256",
      maxmemory_reserved = "2",
      maxmemory_delta    = "2"
      maxmemory_policy   = "allkeys-lru"
    }
}
`

var testAccAzureRMRedisCache_premiumSharded = `
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}

resource "azurerm_redis_cache" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    capacity            = 1
    family              = "P"
    sku_name            = "Premium"
    enable_non_ssl_port = true
    shard_count         = 3
    redis_configuration {
      maxclients         = "256",
      maxmemory_reserved = "2",
      maxmemory_delta    = "2"
      maxmemory_policy   = "allkeys-lru"
    }
}
`

func testAccAzureRMRedisCacheNonStandardCasing(ri int) string {
	return fmt.Sprintf(`
resource "azurerm_resource_group" "test" {
    name = "acctestRG-%d"
    location = "West US"
}
resource "azurerm_redis_cache" "test" {
    name                = "acctestRedis-%d"
    location            = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"
    capacity            = 1
    family              = "c"
    sku_name            = "basic"
    enable_non_ssl_port = false
    redis_configuration {
      maxclients = "256"
    }
}
`, ri, ri)
}
