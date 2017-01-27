package google

import (
	"fmt"
	"log"

	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeRouterPeer() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeRouterPeerCreate,
		Read:   resourceComputeRouterPeerRead,
		Delete: resourceComputeRouterPeerDelete,
		Importer: &schema.ResourceImporter{
			State: resourceComputeRouterPeerImportState,
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
			"interface": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"asn": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
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

func resourceComputeRouterPeerCreate(d *schema.ResourceData, meta interface{}) error {

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
	peerName := d.Get("name").(string)

	routerId := fmt.Sprintf("router/%s/%s", region, routerName)
	mutexKV.Lock(routerId)
	defer mutexKV.Unlock(routerId)

	routersService := compute.NewRoutersService(config.clientCompute)
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router peer because its router %s/%s is gone", region, routerName)
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading router %s/%s: %s", region, routerName, err)
	}

	var peerExists bool = false

	var peers []*compute.RouterBgpPeer = router.BgpPeers
	for _, peer := range peers {

		if peer.Name == peerName {
			peerExists = true
			break
		}
	}

	if !peerExists {

		ifaceName := d.Get("interface").(string)

		peer := &compute.RouterBgpPeer{Name: peerName,
			InterfaceName: ifaceName}

		if v, ok := d.GetOk("ip_address"); ok {
			peer.PeerIpAddress = v.(string)
		}

		if v, ok := d.GetOk("asn"); ok {
			peer.PeerAsn = int64(v.(int))
		}

		log.Printf(
			"[INFO] Adding peer %s", peerName)
		peers = append(peers, peer)
		patchRouter := &compute.Router{
			BgpPeers: peers,
		}

		log.Printf("[DEBUG] Updating router %s/%s with peers: %+v", region, routerName, peers)
		op, err := routersService.Patch(project, region, router.Name, patchRouter).Do()
		if err != nil {
			return fmt.Errorf("Error patching router %s/%s: %s", region, routerName, err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Patching router")
		if err != nil {
			return fmt.Errorf("Error waiting to patch router %s/%s: %s", region, routerName, err)
		}

		d.SetId(fmt.Sprintf("%s/%s/%s", region, routerName, peerName))

	} else {
		log.Printf("[DEBUG] Router %s has peer %s already", routerName, peerName)
	}

	return resourceComputeRouterPeerRead(d, meta)
}

func resourceComputeRouterPeerRead(d *schema.ResourceData, meta interface{}) error {

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
	peerName := d.Get("name").(string)

	routersService := compute.NewRoutersService(config.clientCompute)
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router peer because its router %s/%s is gone", region, routerName)
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading router %s/%s: %s", region, routerName, err)
	}

	var peerFound bool = false

	var peers []*compute.RouterBgpPeer = router.BgpPeers
	for _, peer := range peers {

		if peer.Name == peerName {
			peerFound = true
			d.SetId(fmt.Sprintf("%s/%s/%s", region, routerName, peerName))
			d.Set("interface", peer.InterfaceName)
			d.Set("ip_address", peer.PeerIpAddress)
			d.Set("asn", peer.PeerAsn)
		}
	}
	if !peerFound {
		log.Printf("[WARN] Removing router peer %s/%s/%s because it is gone", region, routerName, peerName)
		d.SetId("")
	}

	return nil
}

func resourceComputeRouterPeerDelete(d *schema.ResourceData, meta interface{}) error {

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
	peerName := d.Get("name").(string)

	routerId := fmt.Sprintf("router/%s/%s", region, routerName)
	mutexKV.Lock(routerId)
	defer mutexKV.Unlock(routerId)

	routersService := compute.NewRoutersService(config.clientCompute)
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router peer because its router %d is gone", d.Get("router").(string))

			return nil
		}

		return fmt.Errorf("Error Reading Router %s: %s", routerName, err)
	}

	var peerFound bool = false

	var oldIfaces []*compute.RouterBgpPeer = router.BgpPeers
	var newIfaces []*compute.RouterBgpPeer = make([]*compute.RouterBgpPeer, len(router.BgpPeers))
	for _, peer := range oldIfaces {

		if peer.Name == peerName {
			peerFound = true
			continue
		} else {
			newIfaces = append(newIfaces, peer)
		}
	}

	if peerFound {

		log.Printf(
			"[INFO] Removing peer %s", peerName)
		patchRouter := &compute.Router{
			BgpPeers: newIfaces,
		}

		log.Printf("[DEBUG] Updating router %s/%s with peers: %+v", region, routerName, newIfaces)
		op, err := routersService.Patch(project, region, router.Name, patchRouter).Do()
		if err != nil {
			return fmt.Errorf("Error patching router %s/%s: %s", region, routerName, err)
		}

		err = computeOperationWaitRegion(config, op, project, region, "Patching router")
		if err != nil {
			return fmt.Errorf("Error waiting to patch router %s/%s: %s", region, routerName, err)
		}

	} else {
		log.Printf("[DEBUG] Router %s/%s had no peer %s already", region, routerName, peerName)
	}

	return nil
}

func resourceComputeRouterPeerImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Invalid router specifier. Expecting {region}/{router}")
	}

	d.Set("region", parts[0])
	d.Set("router", parts[1])
	d.Set("name", parts[2])

	return []*schema.ResourceData{d}, nil
}
