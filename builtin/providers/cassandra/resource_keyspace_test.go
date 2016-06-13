package cassandra

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestSimpleReplicationKeyspace(t *testing.T) {
	var keyspaceMeta gocql.KeyspaceMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKeyspaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSimpleConfig,
				Check: resource.ComposeTestCheckFunc(
					checkKeyspaceExists(testAccSimpleConfigName, &keyspaceMeta),
					checkKeyspaceProperties(&keyspaceMeta, gocql.KeyspaceMetadata{
						Name:            testAccSimpleConfigName,
						DurableWrites:   true,
						StrategyClass:   ReplicationStrategySimple,
						StrategyOptions: map[string]interface{}{"replication_factor": "2"},
					}),
					resource.TestCheckResourceAttr(
						"cassandra_keyspace.test", "name", testAccSimpleConfigName,
					),
					resource.TestCheckResourceAttr(
						"cassandra_keyspace.test", "durable_writes", "true",
					),
					resource.TestCheckResourceAttr(
						"cassandra_keyspace.test", "replication_class", ReplicationStrategySimple,
					),
				),
			},
		},
	})
}

func TestAlterSimpleReplicationKeyspace(t *testing.T) {
	var keyspaceMeta gocql.KeyspaceMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKeyspaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSimpleConfig,
				Check: resource.ComposeTestCheckFunc(
					checkKeyspaceExists(testAccSimpleConfigName, &keyspaceMeta),
					checkKeyspaceProperties(&keyspaceMeta, gocql.KeyspaceMetadata{
						Name:            testAccSimpleConfigName,
						DurableWrites:   true,
						StrategyClass:   ReplicationStrategySimple,
						StrategyOptions: map[string]interface{}{"replication_factor": "2"},
					}),
				),
			},
			resource.TestStep{
				Config: testSimpleConfigTwo,
				Check: resource.ComposeTestCheckFunc(
					checkKeyspaceExists(testAccSimpleConfigName, &keyspaceMeta),
					checkKeyspaceProperties(&keyspaceMeta, gocql.KeyspaceMetadata{
						Name:            testAccSimpleConfigName,
						DurableWrites:   false,
						StrategyClass:   ReplicationStrategySimple,
						StrategyOptions: map[string]interface{}{"replication_factor": "3"},
					}),
				),
			},
		},
	})
}

func TestAlterNetworkReplicationKeyspace(t *testing.T) {
	var keyspaceMeta gocql.KeyspaceMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKeyspaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSimpleConfig,
				Check: resource.ComposeTestCheckFunc(
					checkKeyspaceExists(testAccSimpleConfigName, &keyspaceMeta),
					checkKeyspaceProperties(&keyspaceMeta, gocql.KeyspaceMetadata{
						Name:            testAccSimpleConfigName,
						DurableWrites:   true,
						StrategyClass:   ReplicationStrategySimple,
						StrategyOptions: map[string]interface{}{"replication_factor": "2"},
					}),
				),
			},
			resource.TestStep{
				Config: testNetworkTopologyConfig,
				Check: resource.ComposeTestCheckFunc(
					checkKeyspaceExists(testAccSimpleConfigName, &keyspaceMeta),
					checkKeyspaceProperties(&keyspaceMeta, gocql.KeyspaceMetadata{
						Name:            testAccSimpleConfigName,
						DurableWrites:   false,
						StrategyClass:   ReplicationStrategyNetworkTopology,
						StrategyOptions: map[string]interface{}{"dc1": "2", "dc2": "1"},
					}),
				),
			},
		},
	})
}

/* //TODO: How do we test a failing configuration?
func TestInvalidCreate(t *testing.T) {
	var keyspaceMeta gocql.KeyspaceMetadata

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckKeyspaceDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testInvalidConfig,
				Check: resource.ComposeTestCheckFunc(
					checkKeyspaceExists(testAccSimpleConfigName, &keyspaceMeta),
				),
			},
		},
	})

}
*/

func keyspaceExists(name string) (*gocql.KeyspaceMetadata, error) {
	conn := testAccProvider.Meta().(*gocql.Session)
	return conn.KeyspaceMetadata(name)
}

func checkKeyspaceExists(name string, keyspaceMeta *gocql.KeyspaceMetadata) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		data, err := keyspaceExists(name)
		if err != nil {
			return err
		}
		if data == nil {
			return fmt.Errorf("Keyspace not found: %s", data.Name)
		}
		*keyspaceMeta = *data
		return nil
	}
}

func checkKeyspaceProperties(actualMeta *gocql.KeyspaceMetadata, expectedMeta gocql.KeyspaceMetadata) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expectedMeta.Name != "" && actualMeta.Name != expectedMeta.Name {
			return fmt.Errorf("Keyspace name %s does not match expected %s", actualMeta.Name, expectedMeta.Name)
		}
		if expectedMeta.DurableWrites != actualMeta.DurableWrites {
			return fmt.Errorf("Durable writes %t does not match expected %t", actualMeta.DurableWrites, expectedMeta.DurableWrites)
		}
		// We use Contains, because the actual class looks more like this: 'org.apache.cassandra.locator.SimpleStrategy'
		if expectedMeta.StrategyClass != "" && !strings.Contains(actualMeta.StrategyClass, expectedMeta.StrategyClass) {
			return fmt.Errorf("StrategyClass %s does not match expected %s", actualMeta.StrategyClass, expectedMeta.StrategyClass)
		}
		for key, _ := range expectedMeta.StrategyOptions {
			if key == "class" { // Already checked
				continue
			}
			if expectedMeta.StrategyOptions[key] != actualMeta.StrategyOptions[key] {
				return fmt.Errorf("Strategy options %v did not match expected string: `%v`",
					actualMeta.StrategyOptions[key],
					expectedMeta.StrategyOptions[key],
				)
			}
		}

		return nil
	}
}

func testAccCheckKeyspaceDestroy(s *terraform.State) error {
	time.Sleep(time.Second * time.Duration(2)) // JANK

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cassandra_keyspace" {
			continue
		}

		data, err := keyspaceExists(rs.Primary.ID)

		fmt.Printf("%v\n", data)
		// Gocql returns meta data regardless of existence
		if err == nil && data.StrategyOptions != nil && len(data.StrategyOptions) != 0 {
			return fmt.Errorf("Keyspace %s still exists", rs.Primary.ID)
		}

		if err != nil {
			if !strings.Contains(err.Error(), "not found") {
				return fmt.Errorf("Unexpected error: %s", err)
			}
		}
	}

	return nil
}

const (
	// FYI, gocql and Cassandra 3.5 react oddly to mixed-case keyspace names
	testAccSimpleConfigName = "terraformtest"
	testAccSimpleConfig     = `

resource "cassandra_keyspace" "test" {
    name = "` + testAccSimpleConfigName + `"
    durable_writes = true
    replication_class = "` + ReplicationStrategySimple + `"
    replication_factor = 2
}

`
	testSimpleConfigTwo = `

resource "cassandra_keyspace" "test" {
    name = "` + testAccSimpleConfigName + `"
    durable_writes = false
    replication_class = "` + ReplicationStrategySimple + `"
    replication_factor = 3
}

`
	testNetworkTopologyConfig = `

resource "cassandra_keyspace" "test" {
    name = "` + testAccSimpleConfigName + `"
    durable_writes = false
    replication_class = "` + ReplicationStrategyNetworkTopology + `"
    datacenters {
    	dc1 = 2
    	dc2 = 1
    }
}

`
	testInvalidConfig = `

resource "cassandra_keyspace" "test" {
    name = "` + testAccSimpleConfigName + `"
    durable_writes = false
    replication_class = "` + ReplicationStrategySimple + `"
}

`
)
