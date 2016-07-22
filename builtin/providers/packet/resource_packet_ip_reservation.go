package packet

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/packethost/packngo"
)

func resourcePacketIPReservation() *schema.Resource {
	return &schema.Resource{
		Create: resourcePacketIPReservationCreate,
		Read:   resourcePacketIPReservationRead,
		Delete: resourcePacketIPReservationDelete,

		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				Optional: true,
				Computed: true,
			},

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"network": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"family": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"netmask": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"cidr": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"public": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"assigned_to": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
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

func resourcePacketIPReservationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	createRequest := &packngo.IPReservationRequest{
		Type:     d.Get("type").(string),
		Quantity: d.Get("quantity").(int),
		Comments: d.Get("comments").(string),
	}

	project_id := ""
	if attr, ok := d.GetOk("project_id"); ok {
		project_id = attr.(string)
	}

	newIPReservation, _, err := client.IpReservations.RequestMore(project_id, createRequest)
	if err != nil {
		return friendlyError(err)
	}

	d.SetId(newIPReservation.ID)

	return resourcePacketIPReservationRead(d, meta)
}

func resourcePacketIPReservationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	ip_reservation, _, err := client.IpReservations.Get(d.Id())
	if err != nil {
		err = friendlyError(err)

		// If the ip_reservation somehow already destroyed, mark as succesfully gone.
		if isNotFound(err) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("address", ip_reservation.Address)
	d.Set("network", ip_reservation.Network)
	d.Set("family", ip_reservation.AddressFamily)
	d.Set("netmask", ip_reservation.Netmask)
	d.Set("public", ip_reservation.Public)
	d.Set("cidr", ip_reservation.Cidr)
	d.Set("management", ip_reservation.Management)
	d.Set("manageable", ip_reservation.Manageable)
	d.Set("addon", ip_reservation.Addon)
	d.Set("bill", ip_reservation.Bill)
	d.Set("assignments", ip_reservation.Assignments)
	d.Set("created", ip_reservation.Created)
	d.Set("updated", ip_reservation.Updated)

	return nil
}

func resourcePacketIPReservationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	if _, err := client.IpReservations.Remove(d.Id()); err != nil {
		return friendlyError(err)
	}

	return nil
}
