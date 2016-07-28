package packet

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/packethost/packngo"
)

func resourcePacketIPAddress() *schema.Resource {
	return &schema.Resource{
		Create: resourcePacketIPAddressCreate,
		Read:   resourcePacketIPAddressRead,
		Delete: resourcePacketIPAddressDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"created": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"updated": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePacketIPAddressCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	createRequest := &packngo.IPAddressAssignRequest{
		Address: d.Get("address").(string),
	}

	device_id := ""
	if attr, ok := d.GetOk("instance_id"); ok {
		device_id = attr.(string)
	}

	newIPAddress, _, err := client.Ips.Assign(device_id, createRequest)
	if err != nil {
		return friendlyError(err)
	}

	d.SetId(newIPAddress.ID)

	return resourcePacketIPAddressRead(d, meta)
}

func resourcePacketIPAddressRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	ip_address, _, err := client.Ips.Get(d.Id())
	if err != nil {
		err = friendlyError(err)

		// If the ip_address somehow already destroyed, mark as succesfully gone.
		if isNotFound(err) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("address", ip_address.Address)
	d.Set("gateway", ip_address.Gateway)
	d.Set("network", ip_address.Network)
	d.Set("family", ip_address.AddressFamily)
	d.Set("netmask", ip_address.Netmask)
	d.Set("public", ip_address.Public)
	d.Set("cidr", ip_address.Cidr)
	d.Set("assigned_to", ip_address.AssignedTo)
	d.Set("created", ip_address.Created)
	d.Set("updated", ip_address.Updated)

	return nil
}

func resourcePacketIPAddressDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	if _, err := client.Ips.Unassign(d.Id()); err != nil {
		return friendlyError(err)
	}

	return nil
}
