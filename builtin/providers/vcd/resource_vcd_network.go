package vcd

import (
	"log"

	"bytes"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hmrc/vmware-govcd"
	types "github.com/hmrc/vmware-govcd/types/v56"
	"strings"
)

func resourceVcdNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdNetworkCreate,
		Update: resourceVcdNetworkUpdate,
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
				Default:  "natRouted",
			},

			"edge_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"netmask": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "255.255.255.0",
			},

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"dns1": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "8.8.8.8",
			},

			"dns2": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "8.8.4.4",
			},

			"dns_suffix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"href": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"dhcp_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
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
				Set: resourceVcdNetworkIpAddressHash,
			},
			"static_ip_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
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
				Set: resourceVcdNetworkIpAddressHash,
			},
		},
	}
}

func resourceVcdNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	log.Printf("[TRACE] CLIENT: %#v", vcd_client)
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()

	edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	ipRanges, err := expandIpRange(d.Get("static_ip_pool").(*schema.Set).List())
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}

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

	err = retryCall(4, func() error {
		return vcd_client.OrgVdc.CreateOrgVDCNetwork(newnetwork)
	})
	if err != nil {
		return fmt.Errorf("Error: %#v", err)
	}

	err = vcd_client.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	network, err := vcd_client.OrgVdc.FindVDCNetwork(d.Get("name").(string))
	if err != nil {
		return fmt.Errorf("Error finding network: %#v", err)
	}

	if dhcp, ok := d.GetOk("dhcp_pool"); ok {
		err = retryCall(4, func() error {
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

func resourceVcdNetworkUpdate(d *schema.ResourceData, meta interface{}) error {

	vcd_client := meta.(*govcd.VCDClient)

	log.Printf("[DEBUG] VCD Client configuration: %#v", vcd_client)
	return nil
}

func resourceVcdNetworkRead(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	log.Printf("[DEBUG] VCD Client configuration: %#v", vcd_client)
	log.Printf("[DEBUG] VCD Client configuration: %#v", vcd_client.OrgVdc)

	err := vcd_client.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	network, err := vcd_client.OrgVdc.FindVDCNetwork(d.Id())
	if err != nil {
		log.Printf("[DEBUG] Network no longer exists. Removing from tfstate")
		d.SetId("")
		return nil
	}

	d.Set("name", network.OrgVDCNetwork.Name)
	d.Set("href", network.OrgVDCNetwork.HREF)
	d.Set("fence_mode", network.OrgVDCNetwork.Configuration.FenceMode)
	d.Set("gateway", network.OrgVDCNetwork.Configuration.IPScopes.IPScope.Gateway)
	d.Set("netmask", network.OrgVDCNetwork.Configuration.IPScopes.IPScope.Netmask)
	d.Set("dns1", network.OrgVDCNetwork.Configuration.IPScopes.IPScope.DNS1)
	d.Set("dns2", network.OrgVDCNetwork.Configuration.IPScopes.IPScope.DNS2)

	return nil
}

func resourceVcdNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()
	err := vcd_client.OrgVdc.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing vdc: %#v", err)
	}

	network, err := vcd_client.OrgVdc.FindVDCNetwork(d.Id())
	if err != nil {
		return fmt.Errorf("Error finding network: %#v", err)
	}

	err = retryCall(4, func() error {
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

func resourceVcdNetworkIpAddressHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["start_address"].(string))))
	buf.WriteString(fmt.Sprintf("%s-",
		strings.ToLower(m["end_address"].(string))))

	return hashcode.String(buf.String())
}
