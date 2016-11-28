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

func TestAccLogicalInterconnectGroup_1(t *testing.T) {
	var logicalInterconnectGroup ov.LogicalInterconnectGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLogicalInterconnectGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLogicalInterconnectGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLogicalInterconnectGroupExists(
						"oneview_logical_interconnect_group.test", &logicalInterconnectGroup),
					resource.TestCheckResourceAttr(
						"oneview_logical_interconnect_group.test", "name", "terraform lig",
					),
				),
			},
		},
	})
}

func testAccCheckLogicalInterconnectGroupExists(n string, logicalInterconnectGroup *ov.LogicalInterconnectGroup) resource.TestCheckFunc {
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

		testLogicalInterconnectGroup, err := config.ovClient.GetLogicalInterconnectGroupByName(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testLogicalInterconnectGroup.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*logicalInterconnectGroup = testLogicalInterconnectGroup
		return nil
	}
}

func testAccCheckLogicalInterconnectGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_logical_interconnect_group" {
			continue
		}

		testLig, _ := config.ovClient.GetLogicalInterconnectGroupByName(rs.Primary.ID)

		if testLig.Name != "" {
			return fmt.Errorf("LogicalInterconenctGroup still exists")
		}
	}

	return nil
}

var testAccLogicalInterconnectGroup = `resource "oneview_logical_interconnect_group" "test" {
    count = 1
    name = "terraform lig"
    interconnect_settings {}
    quality_of_service {}
    snmp_configuration {}
    telemetry_configuration {}
  }`
