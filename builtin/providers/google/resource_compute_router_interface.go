package google

import (
	"fmt"
	"log"

	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeRouterInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeRouterInterfaceCreate,
		Read:   resourceComputeRouterInterfaceRead,
		Delete: resourceComputeRouterInterfaceDelete,
		Importer: &schema.ResourceImporter{
			State: resourceComputeRouterInterfaceImportState,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"router": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vpn_tunnel": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_range": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeRouterInterfaceCreate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	routerName := d.Get("router").(string)
	ifaceName := d.Get("name").(string)

	routerId := fmt.Sprintf("router/%s/%s", region, routerName)
	mutexKV.Lock(routerId)
	defer mutexKV.Unlock(routerId)

	routersService := compute.NewRoutersService(config.clientCompute)
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router interface because its router %s/%s is gone", region, routerName)
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading router %s/%s: %s", region, routerName, err)
	}

	var ifaceExists bool = false

	var ifaces []*compute.RouterInterface = router.Interfaces
	for _, iface := range ifaces {

		if iface.Name == ifaceName {
			ifaceExists = true
			break
		}
	}

	if !ifaceExists {

		vpnTunnel, err := getVpnTunnelLink(config, project, region, d.Get("vpn_tunnel").(string))
		if err != nil {
			return err
		}

		iface := &compute.RouterInterface{Name: ifaceName,
			LinkedVpnTunnel: vpnTunnel}

		if v, ok := d.GetOk("ip_range"); ok {
			iface.IpRange = v.(string)
		}

		log.Printf(
			"[INFO] Adding interface %s", ifaceName)
		ifaces = append(ifaces, iface)
		patchRouter := &compute.Router{
			Interfaces: ifaces,
		}

		log.Printf("[DEBUG] Updating router %s/%s with interfaces: %+v", region, routerName, ifaces)
		op, err := routersService.Patch(project, region, router.Name, patchRouter).Do()
		if err != nil {
			return fmt.Errorf("Error patching router %s/%s: %s", region, routerName, err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Patching router")
		if err != nil {
			return fmt.Errorf("Error waiting to patch router %s/%s: %s", region, routerName, err)
		}

		d.SetId(fmt.Sprintf("%s/%s/%s", region, routerName, ifaceName))

	} else {
		log.Printf("[DEBUG] Router %s has interface %s already", routerName, ifaceName)
	}

	return resourceComputeRouterInterfaceRead(d, meta)
}

func resourceComputeRouterInterfaceRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	routerName := d.Get("router").(string)
	ifaceName := d.Get("name").(string)

	routersService := compute.NewRoutersService(config.clientCompute)
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router interface because its router %s/%s is gone", region, routerName)
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading router %s/%s: %s", region, routerName, err)
	}

	var ifaceFound bool = false

	var ifaces []*compute.RouterInterface = router.Interfaces
	for _, iface := range ifaces {

		if iface.Name == ifaceName {
			ifaceFound = true
			d.SetId(fmt.Sprintf("%s/%s/%s", region, routerName, ifaceName))
			// if we don't have a tunnel (when importing), set it to the URI returned from the server
			if _, ok := d.GetOk("vpn_tunnel"); !ok {
				vpnTunnelName, err := getVpnTunnelName(iface.LinkedVpnTunnel)
				if err != nil {
					return err
				}
				d.Set("vpn_tunnel", vpnTunnelName)
			}
			d.Set("ip_range", iface.IpRange)
		}
	}
	if !ifaceFound {
		log.Printf("[WARN] Removing router interface %s/%s/%s because it is gone", region, routerName, ifaceName)
		d.SetId("")
	}

	return nil
}

func resourceComputeRouterInterfaceDelete(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	routerName := d.Get("router").(string)
	ifaceName := d.Get("name").(string)

	routerId := fmt.Sprintf("router/%s/%s", region, routerName)
	mutexKV.Lock(routerId)
	defer mutexKV.Unlock(routerId)

	routersService := compute.NewRoutersService(config.clientCompute)
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router interface because its router %d is gone", d.Get("router").(string))

			return nil
		}

		return fmt.Errorf("Error Reading Router %s: %s", routerName, err)
	}

	var ifaceFound bool = false

	var oldIfaces []*compute.RouterInterface = router.Interfaces
	var newIfaces []*compute.RouterInterface = make([]*compute.RouterInterface, len(router.Interfaces))
	for _, iface := range oldIfaces {

		if iface.Name == ifaceName {
			ifaceFound = true
			continue
		} else {
			newIfaces = append(newIfaces, iface)
		}
	}

	if ifaceFound {

		log.Printf(
			"[INFO] Removing interface %s", ifaceName)
		patchRouter := &compute.Router{
			Interfaces: newIfaces,
		}

		log.Printf("[DEBUG] Updating router %s/%s with interfaces: %+v", region, routerName, newIfaces)
		op, err := routersService.Patch(project, region, router.Name, patchRouter).Do()
		if err != nil {
			return fmt.Errorf("Error patching router %s/%s: %s", region, routerName, err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Patching router")
		if err != nil {
			return fmt.Errorf("Error waiting to patch router %s/%s: %s", region, routerName, err)
		}

	} else {
		log.Printf("[DEBUG] Router %s/%s had no interface %s already", region, routerName, ifaceName)
	}

	return nil
}

func resourceComputeRouterInterfaceImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Invalid router specifier. Expecting {region}/{router}")
	}

	d.Set("region", parts[0])
	d.Set("router", parts[1])
	d.Set("name", parts[2])

	return []*schema.ResourceData{d}, nil
}
