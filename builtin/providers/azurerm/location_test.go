package azurerm

import "testing"

func TestAzureRMNormalizeLocation(t *testing.T) {
	s := azureRMNormalizeLocation("West US")
	if s != "westus" {
		t.Fatalf("expected location to equal westus, actual %s", s)
	}
}
