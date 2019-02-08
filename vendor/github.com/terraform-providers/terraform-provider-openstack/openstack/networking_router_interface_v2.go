package openstack

import (
	"log"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceNetworkingRouterInterfaceV2StateRefreshFunc(networkingClient *gophercloud.ServiceClient, portID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		r, err := ports.Get(networkingClient, portID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return r, "DELETED", nil
			}

			return r, "", err
		}

		return r, r.Status, nil
	}
}

func resourceNetworkingRouterInterfaceV2DeleteRefreshFunc(networkingClient *gophercloud.ServiceClient, d *schema.ResourceData) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		routerID := d.Get("router_id").(string)
		routerInterfaceID := d.Id()

		log.Printf("[DEBUG] Attempting to delete openstack_networking_router_interface_v2 %s", routerInterfaceID)

		removeOpts := routers.RemoveInterfaceOpts{
			SubnetID: d.Get("subnet_id").(string),
			PortID:   d.Get("port_id").(string),
		}

		r, err := ports.Get(networkingClient, routerInterfaceID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted openstack_networking_router_interface_v2 %s", routerInterfaceID)
				return r, "DELETED", nil
			}
			return r, "ACTIVE", err
		}

		_, err = routers.RemoveInterface(networkingClient, routerID, removeOpts).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted openstack_networking_router_interface_v2 %s", routerInterfaceID)
				return r, "DELETED", nil
			}
			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					log.Printf("[DEBUG] openstack_networking_router_interface_v2 %s is still in use", routerInterfaceID)
					return r, "ACTIVE", nil
				}
			}

			return r, "ACTIVE", err
		}

		log.Printf("[DEBUG] openstack_networking_router_interface_v2 %s is still active", routerInterfaceID)
		return r, "ACTIVE", nil
	}
}
