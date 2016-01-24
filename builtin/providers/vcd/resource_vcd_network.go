package vcd

import (
	"log"

	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	types "github.com/hmrc/vmware-govcd/types/v56"
)

func resourceVcdNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdNetworkCreate,
		Read:   resourceVcdNetworkRead,
		Delete: resourceVcdNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"fence_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "natRouted",
			},

			"edge_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"netmask": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "255.255.255.0",
			},

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"dns1": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "8.8.8.8",
			},

			"dns2": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "8.8.4.4",
			},

			"dns_suffix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"href": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"dhcp_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"end_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceVcdNetworkIPAddressHash,
			},
			"static_ip_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"end_address": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceVcdNetworkIPAddressHash,
			},
		},
	}
}

func resourceVcdNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	log.Printf("[TRACE] CLIENT: %#v", vcdClient)
	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()

	edgeGateway, err := vcdClient.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	ipRanges := expandIPRange(d.Get("static_ip_pool").(*schema.Set).List())

	newnetwork := &types.OrgVDCNetwork{
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		Name:  d.Get("name").(string),
		Configuration: &types.NetworkConfiguration{
			FenceMode: d.Get("fence_mode").(string),
			IPScopes: &types.IPScopes{
				IPScope: types.IPScope{
					IsInherited: false,
					Gateway:     d.Get("gateway").(string),
					Netmask:     d.Get("netmask").(string),
					DNS1:        d.Get("dns1").(string),
					DNS2:        d.Get("dns2").(string),
					DNSSuffix:   d.Get("dns_suffix").(string),
					IPRanges:    &ipRanges,
				},
			},
			BackwardCompatibilityMode: true,
		},
		EdgeGateway: &types.Reference{
			HREF: edgeGateway.EdgeGateway.HREF,
		},
		IsShared: false,
	}

	log.Printf("[INFO] NETWORK: %#v", newnetwork)

	err = retryCall(vcdClient.MaxRetryTimeout, func() error {
		return vcdClient.OrgVdc.CreateOrgVDCNetwork(newnetwork)
	})
	if err != nil {
		return fmt.Errorf("Error: %#v", err)
	}

	err = vcdClient.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	network, err := vcdClient.OrgVdc.FindVDCNetwork(d.Get("name").(string))
	if err != nil {
		return fmt.Errorf("Error finding network: %#v", err)
	}

	if dhcp, ok := d.GetOk("dhcp_pool"); ok {
		err = retryCall(vcdClient.MaxRetryTimeout, func() error {
			task, err := edgeGateway.AddDhcpPool(network.OrgVDCNetwork, dhcp.(*schema.Set).List())
			if err != nil {
				return fmt.Errorf("Error adding DHCP pool: %#v", err)
			}

			return task.WaitTaskCompletion()
		})
		if err != nil {
			return fmt.Errorf("Error completing tasks: %#v", err)
		}

	}

	d.SetId(d.Get("name").(string))

	return resourceVcdNetworkRead(d, meta)
}

func resourceVcdNetworkRead(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	log.Printf("[DEBUG] VCD Client configuration: %#v", vcdClient)
	log.Printf("[DEBUG] VCD Client configuration: %#v", vcdClient.OrgVdc)

	err := vcdClient.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	network, err := vcdClient.OrgVdc.FindVDCNetwork(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Network no longer exists. Removing from tfstate")
		d.SetId("")
		return nil
	}

	d.Set("name", network.OrgVDCNetwork.Name)
	d.Set("href", network.OrgVDCNetwork.HREF)
	if c := network.OrgVDCNetwork.Configuration; c != nil {
		d.Set("fence_mode", c.FenceMode)
		if c.IPScopes != nil {
			d.Set("gateway", c.IPScopes.IPScope.Gateway)
			d.Set("netmask", c.IPScopes.IPScope.Netmask)
			d.Set("dns1", c.IPScopes.IPScope.DNS1)
			d.Set("dns2", c.IPScopes.IPScope.DNS2)
		}
	}

	return nil
}

func resourceVcdNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()
	err := vcdClient.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	network, err := vcdClient.OrgVdc.FindVDCNetwork(d.Id())
	if err != nil {
		return fmt.Errorf("Error finding network: %#v", err)
	}

	err = retryCall(vcdClient.MaxRetryTimeout, func() error {
		task, err := network.Delete()
		if err != nil {
			return fmt.Errorf("Error Deleting Network: %#v", err)
		}
		return task.WaitTaskCompletion()
	})
	if err != nil {
		return err
	}

	return nil
}

func resourceVcdNetworkIPAddressHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["start_address"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["end_address"].(string))))

	return hashcode.String(buf.String())
}
