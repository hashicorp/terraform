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

func TestAccI3SPlan_1(t *testing.T) {
	var server ov.ServerProfile

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckI3SPlanDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccI3SPlan,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckI3SPlanExists(
						"oneview_i3s_plan.test", &server),
					resource.TestCheckResourceAttr(
						"oneview_i3s_plan.test", "name", "Terraform Server 1",
					),
				),
			},
		},
	})
}

func testAccCheckI3SPlanExists(n string, server *ov.ServerProfile) resource.TestCheckFunc {
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

		testServer, err := config.ovClient.GetProfileByName(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testServer.Name != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*server = testServer
		return nil
	}
}

func testAccCheckI3SPlanDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_i3s_plan" {
			continue
		}

		testServer, _ := config.ovClient.GetProfileByName(rs.Primary.ID)

		if !testServer.OSDeploymentSettings.OSDeploymentPlanUri.IsNil() {
			return fmt.Errorf("Deployment Plan still exists")
		}
	}

	return nil
}

var testAccI3SPlan = `resource "oneview_server_profile" "test" {
     count           = 1
     name            = "terraform-test-${count.index}"
     template = "Matthew-No-I3S"
     hardware_name        = "Frame 0, bay 7"
     type = "ServerProfileV6"
   }

   resource "oneview_i3s_plan" "test" {
     count = 1
     server_name = "${oneview_server_profile.test.name}"
     os_deployment_plan = "dennis_dp_empty_volume"
   }

  `
