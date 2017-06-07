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

			"peer_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"peer_asn": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"advertised_route_priority": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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

	routerLock := getRouterLockName(region, routerName)
	mutexKV.Lock(routerLock)
	defer mutexKV.Unlock(routerLock)

	routersService := config.clientCompute.Routers
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router peer %s because its router %s/%s is gone", peerName, region, routerName)
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading router %s/%s: %s", region, routerName, err)
	}

	peers := router.BgpPeers
	for _, peer := range peers {
		if peer.Name == peerName {
			d.SetId("")
			return fmt.Errorf("Router %s has peer %s already", routerName, peerName)
		}
	}

	ifaceName := d.Get("interface").(string)

	peer := &compute.RouterBgpPeer{Name: peerName,
		InterfaceName: ifaceName}

	if v, ok := d.GetOk("peer_ip_address"); ok {
		peer.PeerIpAddress = v.(string)
	}

	if v, ok := d.GetOk("peer_asn"); ok {
		peer.PeerAsn = int64(v.(int))
	}

	if v, ok := d.GetOk("advertised_route_priority"); ok {
		peer.AdvertisedRoutePriority = int64(v.(int))
	}

	log.Printf("[INFO] Adding peer %s", peerName)
	peers = append(peers, peer)
	patchRouter := &compute.Router{
		BgpPeers: peers,
	}

	log.Printf("[DEBUG] Updating router %s/%s with peers: %+v", region, routerName, peers)
	op, err := routersService.Patch(project, region, router.Name, patchRouter).Do()
	if err != nil {
		return fmt.Errorf("Error patching router %s/%s: %s", region, routerName, err)
	}
	d.SetId(fmt.Sprintf("%s/%s/%s", region, routerName, peerName))
	err = computeOperationWaitRegion(config, op, project, region, "Patching router")
	if err != nil {
		d.SetId("")
		return fmt.Errorf("Error waiting to patch router %s/%s: %s", region, routerName, err)
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

	routersService := config.clientCompute.Routers
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router peer %s because its router %s/%s is gone", peerName, region, routerName)
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading router %s/%s: %s", region, routerName, err)
	}

	for _, peer := range router.BgpPeers {

		if peer.Name == peerName {
			d.SetId(fmt.Sprintf("%s/%s/%s", region, routerName, peerName))
			d.Set("interface", peer.InterfaceName)
			d.Set("peer_ip_address", peer.PeerIpAddress)
			d.Set("peer_asn", peer.PeerAsn)
			d.Set("advertised_route_priority", peer.AdvertisedRoutePriority)
			d.Set("ip_address", peer.IpAddress)
			d.Set("region", region)
			d.Set("project", project)
			return nil
		}
	}

	log.Printf("[WARN] Removing router peer %s/%s/%s because it is gone", region, routerName, peerName)
	d.SetId("")
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

	routerLock := getRouterLockName(region, routerName)
	mutexKV.Lock(routerLock)
	defer mutexKV.Unlock(routerLock)

	routersService := config.clientCompute.Routers
	router, err := routersService.Get(project, region, routerName).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router peer %s because its router %s/%s is gone", peerName, region, routerName)

			return nil
		}

		return fmt.Errorf("Error Reading Router %s: %s", routerName, err)
	}

	var newPeers []*compute.RouterBgpPeer = make([]*compute.RouterBgpPeer, 0, len(router.BgpPeers))
	for _, peer := range router.BgpPeers {
		if peer.Name == peerName {
			continue
		} else {
			newPeers = append(newPeers, peer)
		}
	}

	if len(newPeers) == len(router.BgpPeers) {
		log.Printf("[DEBUG] Router %s/%s had no peer %s already", region, routerName, peerName)
		d.SetId("")
		return nil
	}

	log.Printf(
		"[INFO] Removing peer %s from router %s/%s", peerName, region, routerName)
	patchRouter := &compute.Router{
		BgpPeers: newPeers,
	}

	log.Printf("[DEBUG] Updating router %s/%s with peers: %+v", region, routerName, newPeers)
	op, err := routersService.Patch(project, region, router.Name, patchRouter).Do()
	if err != nil {
		return fmt.Errorf("Error patching router %s/%s: %s", region, routerName, err)
	}

	err = computeOperationWaitRegion(config, op, project, region, "Patching router")
	if err != nil {
		return fmt.Errorf("Error waiting to patch router %s/%s: %s", region, routerName, err)
	}

	d.SetId("")
	return nil
}

func resourceComputeRouterPeerImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Invalid router peer specifier. Expecting {region}/{router}/{peer}")
	}

	d.Set("region", parts[0])
	d.Set("router", parts[1])
	d.Set("name", parts[2])

	return []*schema.ResourceData{d}, nil
}
