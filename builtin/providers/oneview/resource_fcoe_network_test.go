// (C) Copyright 2016 Hewlett Packard Enterprise Development LP
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package oneview

import (
	"fmt"
	"testing"

	"github.com/HewlettPackard/oneview-golang/ov"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccFCoENetwork_1(t *testing.T) {
	var fcoeNetwork ov.FCoENetwork

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckFCoENetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccFCoENetwork,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFCoENetworkExists(
						"oneview_fcoe_network.test", &fcoeNetwork),
					resource.TestCheckResourceAttr(
						"oneview_fcoe_network.test", "name", "Terraform FCoE Network 1",
					),
				),
			},
		},
	})
}

func testAccCheckFCoENetworkExists(n string, fcoeNetwork *ov.FCoENetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found :%v", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No ID is set")
		}

		config, err := testProviderConfig()
		if err != nil {
			return err
		}

		testFCoENetwork, err := config.ovClient.GetFCoENetworkByName(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testFCoENetwork.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*fcoeNetwork = testFCoENetwork
		return nil
	}
}

func testAccCheckFCoENetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_fcoe_network" {
			continue
		}

		testFCoENet, _ := config.ovClient.GetFCoENetworkByName(rs.Primary.ID)

		if testFCoENet.Name != "" {
			return fmt.Errorf("FCoENetwork still exists")
		}
	}

	return nil
}

var testAccFCoENetwork = `
  resource "oneview_fcoe_network" "test" {
    name = "Terraform FCoE Network 1"
    vlanId = 157
  }`
