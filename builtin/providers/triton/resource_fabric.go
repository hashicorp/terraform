package triton

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/gosdc/cloudapi"
)

func resourceFabric() *schema.Resource {
	return &schema.Resource{
		Create: resourceFabricCreate,
		Exists: resourceFabricExists,
		Read:   resourceFabricRead,
		Delete: resourceFabricDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "network name",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"public": {
				Description: "whether or not this is an RFC1918 network",
				Computed:    true,
				Type:        schema.TypeBool,
			},
			"fabric": {
				Description: "whether or not this network is on a fabric",
				Computed:    true,
				Type:        schema.TypeBool,
			},
			"description": {
				Description: "optional description of network",
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"subnet": {
				Description: "CIDR formatted string describing network",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"provision_start_ip": {
				Description: "first IP on the network that can be assigned",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"provision_end_ip": {
				Description: "last assignable IP on the network",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"gateway": {
				Description: "optional gateway IP",
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"resolvers": {
				Description: "array of IP addresses for resolvers",
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"routes": {
				Description: "map of CIDR block to Gateway IP address",
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeMap,
			},
			"internet_nat": {
				Description: "if a NAT zone is provisioned at Gateway IP address",
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeBool,
			},
			"vlan_id": {
				Description: "VLAN network is on",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeInt,
			},
		},
	}
}

func resourceFabricCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	var resolvers []string
	for _, resolver := range d.Get("resolvers").([]interface{}) {
		resolvers = append(resolvers, resolver.(string))
	}

	routes := map[string]string{}
	for cidr, v := range d.Get("routes").(map[string]interface{}) {
		ip, ok := v.(string)
		if !ok {
			return fmt.Errorf(`cannot use "%v" as an IP address`, v)
		}
		routes[cidr] = ip
	}

	fabric, err := client.CreateFabricNetwork(
		int16(d.Get("vlan_id").(int)),
		cloudapi.CreateFabricNetworkOpts{
			Name:             d.Get("name").(string),
			Description:      d.Get("description").(string),
			Subnet:           d.Get("subnet").(string),
			ProvisionStartIp: d.Get("provision_start_ip").(string),
			ProvisionEndIp:   d.Get("provision_end_ip").(string),
			Gateway:          d.Get("gateway").(string),
			Resolvers:        resolvers,
			Routes:           routes,
			InternetNAT:      d.Get("internet_nat").(bool),
		},
	)
	if err != nil {
		return err
	}

	d.SetId(fabric.Id)

	err = resourceFabricRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceFabricExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*cloudapi.Client)

	fabric, err := client.GetFabricNetwork(int16(d.Get("vlan_id").(int)), d.Id())

	return fabric != nil && err == nil, err
}

func resourceFabricRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	fabric, err := client.GetFabricNetwork(int16(d.Get("vlan_id").(int)), d.Id())
	if err != nil {
		return err
	}

	d.SetId(fabric.Id)
	d.Set("name", fabric.Name)
	d.Set("public", fabric.Public)
	d.Set("public", fabric.Public)
	d.Set("fabric", fabric.Fabric)
	d.Set("description", fabric.Description)
	d.Set("subnet", fabric.Subnet)
	d.Set("provision_start_ip", fabric.ProvisionStartIp)
	d.Set("provision_end_ip", fabric.ProvisionEndIp)
	d.Set("gateway", fabric.Gateway)
	d.Set("resolvers", fabric.Resolvers)
	d.Set("routes", fabric.Routes)
	d.Set("internet_nat", fabric.InternetNAT)
	d.Set("vlan_id", fabric.VLANId)

	return nil
}

func resourceFabricDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudapi.Client)

	return client.DeleteFabricNetwork(int16(d.Get("vlan_id").(int)), d.Id())
}
