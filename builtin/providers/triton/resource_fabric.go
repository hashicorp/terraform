package triton

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/joyent/triton-go"
)

func resourceFabric() *schema.Resource {
	return &schema.Resource{
		Create: resourceFabricCreate,
		Exists: resourceFabricExists,
		Read:   resourceFabricRead,
		Delete: resourceFabricDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Network name",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"public": {
				Description: "Whether or not this is an RFC1918 network",
				Computed:    true,
				Type:        schema.TypeBool,
			},
			"fabric": {
				Description: "Whether or not this network is on a fabric",
				Computed:    true,
				Type:        schema.TypeBool,
			},
			"description": {
				Description: "Description of network",
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"subnet": {
				Description: "CIDR formatted string describing network address space",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"provision_start_ip": {
				Description: "First IP on the network that can be assigned",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"provision_end_ip": {
				Description: "Last assignable IP on the network",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"gateway": {
				Description: "Gateway IP",
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeString,
			},
			"resolvers": {
				Description: "List of IP addresses for DNS resolvers",
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"routes": {
				Description: "Map of CIDR block to Gateway IP address",
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeMap,
			},
			"internet_nat": {
				Description: "Whether or not a NAT zone is provisioned at the Gateway IP address",
				Computed:    true,
				Optional:    true,
				ForceNew:    true,
				Type:        schema.TypeBool,
			},
			"vlan_id": {
				Description: "VLAN on which the network exists",
				Required:    true,
				ForceNew:    true,
				Type:        schema.TypeInt,
			},
		},
	}
}

func resourceFabricCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	var resolvers []string
	for _, resolver := range d.Get("resolvers").([]interface{}) {
		resolvers = append(resolvers, resolver.(string))
	}

	routes := map[string]string{}
	for cidr, v := range d.Get("routes").(map[string]interface{}) {
		ip, ok := v.(string)
		if !ok {
			return fmt.Errorf(`Cannot use "%v" as an IP address`, v)
		}
		routes[cidr] = ip
	}

	fabric, err := client.Fabrics().CreateFabricNetwork(context.Background(), &triton.CreateFabricNetworkInput{
		FabricVLANID:     d.Get("vlan_id").(int),
		Name:             d.Get("name").(string),
		Description:      d.Get("description").(string),
		Subnet:           d.Get("subnet").(string),
		ProvisionStartIP: d.Get("provision_start_ip").(string),
		ProvisionEndIP:   d.Get("provision_end_ip").(string),
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

	return resourceFabricRead(d, meta)
}

func resourceFabricExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*triton.Client)

	return resourceExists(client.Fabrics().GetFabricNetwork(context.Background(), &triton.GetFabricNetworkInput{
		FabricVLANID: d.Get("vlan_id").(int),
		NetworkID:    d.Id(),
	}))
}

func resourceFabricRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	fabric, err := client.Fabrics().GetFabricNetwork(context.Background(), &triton.GetFabricNetworkInput{
		FabricVLANID: d.Get("vlan_id").(int),
		NetworkID:    d.Id(),
	})
	if err != nil {
		return err
	}

	d.SetId(fabric.Id)
	d.Set("name", fabric.Name)
	d.Set("public", fabric.Public)
	d.Set("fabric", fabric.Fabric)
	d.Set("description", fabric.Description)
	d.Set("subnet", fabric.Subnet)
	d.Set("provision_start_ip", fabric.ProvisioningStartIP)
	d.Set("provision_end_ip", fabric.ProvisioningEndIP)
	d.Set("gateway", fabric.Gateway)
	d.Set("resolvers", fabric.Resolvers)
	d.Set("routes", fabric.Routes)
	d.Set("internet_nat", fabric.InternetNAT)
	d.Set("vlan_id", d.Get("vlan_id").(int))

	return nil
}

func resourceFabricDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*triton.Client)

	return client.Fabrics().DeleteFabricNetwork(context.Background(), &triton.DeleteFabricNetworkInput{
		FabricVLANID: d.Get("vlan_id").(int),
		NetworkID:    d.Id(),
	})
}
