package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccComputeRouterInterface_import(t *testing.T) {
	resourceName := "google_compute_router_interface.foobar"
	network := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	subnet := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	address := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	gateway := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	espRule := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	udp500Rule := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	udp4500Rule := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	router := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	tunnel := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	iface := fmt.Sprintf("router-interface-import-test-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccComputeRouterInterfaceBasic(network, subnet, address, gateway, espRule, udp500Rule,
					udp4500Rule, router, tunnel, iface),
			},

			resource.TestStep{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
