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

func TestAccNetworkSet_1(t *testing.T) {
	var networkSet ov.NetworkSet

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkSetDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNetworkSet,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkSetExists(
						"oneview_network_set.test", &networkSet),
					resource.TestCheckResourceAttr(
						"oneview_network_set.test", "name", "Terraform Network Set 1",
					),
				),
			},
		},
	})
}

func testAccCheckNetworkSetExists(n string, networkSet *ov.NetworkSet) resource.TestCheckFunc {
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

		testNetworkSet, err := config.ovClient.GetNetworkSetByName(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testNetworkSet.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*networkSet = testNetworkSet
		return nil
	}
}

func testAccCheckNetworkSetDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_network_set" {
			continue
		}

		testNet, _ := config.ovClient.GetNetworkSetByName(rs.Primary.ID)

		if testNet.Name != "" {
			return fmt.Errorf("NetworkSet still exists")
		}
	}

	return nil
}

var testAccNetworkSet = `
  resource "oneview_network_set" "test" {
    name = "Terraform Network Set 1"
  }`
