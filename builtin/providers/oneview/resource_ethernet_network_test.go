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

func TestAccEthernetNetwork_1(t *testing.T) {
	var ethernetNetwork ov.EthernetNetwork

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckEthernetNetworkDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccEthernetNetwork,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEthernetNetworkExists(
						"oneview_ethernet_network.test", &ethernetNetwork),
					resource.TestCheckResourceAttr(
						"oneview_ethernet_network.test", "name", "Terraform Ethernet Network 1",
					),
				),
			},
		},
	})
}

func testAccCheckEthernetNetworkExists(n string, ethernetNetwork *ov.EthernetNetwork) resource.TestCheckFunc {
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

		testEthernetNetwork, err := config.ovClient.GetEthernetNetworkByName(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testEthernetNetwork.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*ethernetNetwork = testEthernetNetwork
		return nil
	}
}

func testAccCheckEthernetNetworkDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_ethernet_network" {
			continue
		}

		testNet, _ := config.ovClient.GetEthernetNetworkByName(rs.Primary.ID)

		if testNet.Name != "" {
			return fmt.Errorf("EthernetNetwork still exists")
		}
	}

	return nil
}

var testAccEthernetNetwork = `
  resource "oneview_ethernet_network" "test" {
    name = "Terraform Ethernet Network 1"
    vlanId = "${117}"
  }`
