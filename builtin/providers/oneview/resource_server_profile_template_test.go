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

func TestAccServerProfileTemplate_1(t *testing.T) {
	var serverProfileTemplate ov.ServerProfile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckServerProfileTemplateDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServerProfileTemplate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServerProfileTemplateExists(
						"oneview_server_profile_template.test", &serverProfileTemplate),
					resource.TestCheckResourceAttr(
						"oneview_server_profile_template.test", "name", "terraform test spt",
					),
				),
			},
		},
	})
}

func testAccCheckServerProfileTemplateExists(n string, serverProfileTemplate *ov.ServerProfile) resource.TestCheckFunc {
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

		testServerProfileTemplate, err := config.ovClient.GetProfileTemplateByName(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testServerProfileTemplate.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*serverProfileTemplate = testServerProfileTemplate
		return nil
	}
}

func testAccCheckServerProfileTemplateDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_server_profile_template" {
			continue
		}

		testServerProfileTemplate, _ := config.ovClient.GetProfileTemplateByName(rs.Primary.ID)

		if testServerProfileTemplate.Name != "" {
			return fmt.Errorf("ServerProfileTemplate still exists")
		}
	}

	return nil
}

var testAccServerProfileTemplate = `
  resource "oneview_server_profile_template" "test" {
    name = "terraform test spt"
    server_hardware_type = "BL460c Gen9 1"
    enclosure_group = "Houston"
  }`
