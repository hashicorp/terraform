package azurerm

import (
	"fmt"
	"github.com/hashicorp/terraform/terraform"
)

func getArmResourceNameAndGroupByTerraformName(s *terraform.State, name string) (string, string, error) {
	rs, ok := s.RootModule().Resources[name]
	if !ok {
		return "", "", fmt.Errorf("Not found: %s", name)
	}

	armName := rs.Primary.Attributes["name"]
	armResourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
	if !hasResourceGroup {
		return "", "", fmt.Errorf("Bad: no resource group found in state for virtual network gateway: %s", name)
	}

	return armName, armResourceGroup, nil
}
