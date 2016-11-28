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

	"github.com/HewlettPackard/oneview-golang/icsp"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccICSP_Server_1(t *testing.T) {
	var icspServer icsp.Server

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckICSPServerDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccICSPServer,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckICSPServerExists(
						"oneview_icsp_server.test", &icspServer)),
			},
		},
	})
}

func testAccCheckICSPServerExists(n string, icspServer *icsp.Server) resource.TestCheckFunc {
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

		testICSPServer, err := config.icspClient.GetServerByIP(rs.Primary.ID)
		if err != nil {
			return err
		}
		if testICSPServer.ILO.IPAddress != rs.Primary.ID {
			return fmt.Errorf("Instance not found")
		}
		*icspServer = testICSPServer
		return nil
	}
}

func testAccCheckICSPServerDestroy(s *terraform.State) error {
	config := testAccProvider.Meta().(*Config)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "oneview_icsp_server" {
			continue
		}

		testICSPServer, _ := config.icspClient.GetServerByIP(rs.Primary.ID)

		if testICSPServer.Name != "" {
			return fmt.Errorf("ICSP server still exists")
		}
	}

	return nil
}

var testAccICSPServer = `resource "oneview_server_profile" "test" {
     count           = 1
     name            = "terraform-test-${count.index}"
     template = "UCP Template iSCSI"
   }

   resource "oneview_icsp_server" "test" {
     count = 1
     ilo_ip = "${element(oneview_server_profile.test.*.ilo_ip, count.index)}"
     user_name = "ICspUser"
     password = "@utoPr0vi$ion"
     serial_number = "${element(oneview_server_profile.test.*.serial_number, count.index)}"
   }
  `
