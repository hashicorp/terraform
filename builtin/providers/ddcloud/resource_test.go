package ddcloud

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"strings"
	"testing"
)

// Aggregate test helpers for resources.

// Data structure that holds resource data for use between test steps.
type testAccResourceData struct {
	// Map from terraform resource names to provider resource Ids.
	NamesToResourceIDs map[string]string
}

func newTestAccResourceData() testAccResourceData {
	return testAccResourceData{
		NamesToResourceIDs: map[string]string{},
	}
}

// The configuration for a resource-update acceptance test.
type testAccResourceUpdate struct {
	ResourceName  string
	CheckDestroy  resource.TestCheckFunc
	InitialConfig string
	InitialCheck  resource.TestCheckFunc
	UpdateConfig  string
	UpdateCheck   resource.TestCheckFunc
}

// Aggregate test - update resource in-place (resource is updated, not destroyed and re-created).
func testAccResourceUpdateInPlace(test *testing.T, testDefinition testAccResourceUpdate) {
	resourceData := newTestAccResourceData()

	resource.Test(test, resource.TestCase{
		Providers:    testAccProviders,
		CheckDestroy: testDefinition.CheckDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testDefinition.InitialConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckCaptureID(testDefinition.ResourceName, &resourceData),
					testDefinition.InitialCheck,
				),
			},
			resource.TestStep{
				Config: testDefinition.UpdateConfig,
				Check: resource.ComposeTestCheckFunc(
					testCheckResourceUpdatedInPlace(testDefinition.ResourceName, &resourceData),
					testDefinition.UpdateCheck,
				),
			},
		},
	})
}

// Acceptance test check helper:
//
// Capture the resource's Id.
func testCheckCaptureID(name string, testData *testAccResourceData) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		testData.NamesToResourceIDs[name] = res.Primary.ID

		return nil
	}
}

// Acceptance test check helper:
//
// Check if the resource was updated in-place (its Id has not changed).
func testCheckResourceUpdatedInPlace(name string, testData *testAccResourceData) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceType, err := getResourceTypeFromName(name)
		if err != nil {
			return err
		}

		capturedResourceID, ok := testData.NamesToResourceIDs[name]
		if !ok {
			return fmt.Errorf("No Id has been captured for resource '%s'.", name)
		}

		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		currentResourceID := res.Primary.ID
		if currentResourceID != capturedResourceID {
			return fmt.Errorf("Bad: The update was expected to be performed in-place but the Id for %s has changed (was: %s, now: %s) which indicates that the resource was destroyed and re-created", resourceType, capturedResourceID, currentResourceID)
		}

		return nil
	}
}

// Acceptance test check helper:
//
// Check if the resource was updated by destroying and re-creating it (its Id has changed).
func testCheckResourceReplaced(name string, testData *testAccResourceData) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		resourceType, err := getResourceTypeFromName(name)
		if err != nil {
			return err
		}

		capturedResourceID, ok := testData.NamesToResourceIDs[name]
		if !ok {
			return fmt.Errorf("No Id has been captured for resource '%s'.", name)
		}

		res, ok := state.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		currentResourceID := res.Primary.ID
		if currentResourceID == capturedResourceID {
			return fmt.Errorf("Bad: The update was expected to be performed by destroying and re-creating  %s but its Id has changed (was: %s, now: %s) which indicates that the resource was performed in-place", resourceType, capturedResourceID, currentResourceID)
		}

		return nil
	}
}

func getResourceTypeFromName(name string) (string, error) {
	resourceNameComponents := strings.SplitN(name, ".", 2)
	if len(resourceNameComponents) != 2 {
		return "", fmt.Errorf("Invalid resource name: '%s' (should be 'resource_type.resource_name')", name)
	}

	return resourceNameComponents[0], nil
}
