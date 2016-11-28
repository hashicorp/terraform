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

func TestAccLogicalSwitchGroup_1(t *testing.T) {
	var logicalSwitchGroup ov.LogicalSwitchGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckLogicalSwitchGroupDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccLogicalSwitchGroup,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckLogicalSwitchGroupExists(
						"oneview_logical_switch_group.test", &logicalSwitchGroup),
					resource.TestCheckResourceAttr(
						"oneview_logical_switch_group.test", "name", "terraform lsg",
					),
				),
			},
		},
	})
}

func testAccCheckLogicalSwitchGroupExists(n string, logicalSwitchGroup *ov.LogicalSwitchGroup) resource.TestCheckFunc {
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

		testLogicalSwitchGroup, err := config.ovClient.GetLogicalSwitchGroupByName(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testLogicalSwitchGroup.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*logicalSwitchGroup = testLogicalSwitchGroup
		return nil
	}
}

func testAccCheckLogicalSwitchGroupDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_logical_switch_group" {
			continue
		}

		testLSG, _ := config.ovClient.GetLogicalSwitchGroupByName(rs.Primary.ID)

		if testLSG.Name != "" {
			return fmt.Errorf("LogicalSwitchGroup still exists")
		}
	}

	return nil
}

var testAccLogicalSwitchGroup = `resource "oneview_logical_switch_group" "test" {
    count = 1
    name = "terraform lsg"
    switch_type_name = "Cisco Nexus 50xx"
    switch_count = 2
  }`
