package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackStaticRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackStaticRouteCreate,
		Read:   resourceCloudStackStaticRouteRead,
		Delete: resourceCloudStackStaticRouteDelete,

		Schema: map[string]*schema.Schema{
			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			/*"nexthop": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},*/

			"gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackStaticRouteCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	if err := verifyStaticRouteParams(d); err != nil {
		return err
	}

	c := d.Get("cidr").(string)

	// Create a new parameter struct
	p := cs.VPC.NewCreateStaticRouteParams(
		c,
		d.Get("gateway_id").(string),
	)

	// If there is a gateway_id supplied, add it to the parameter struct
	/*if gatewayid, ok := d.GetOk("gateway_id"); ok {
		p.SetGatewayid(gatewayid.(string))
	}*/

	// Create the new private gateway
	r, err := cs.VPC.CreateStaticRoute(p)
	if err != nil {
		return fmt.Errorf("Error creating static route for %s: %s", c, err)
	}

	d.SetId(r.Id)

	return resourceCloudStackStaticRouteRead(d, meta)
}

func resourceCloudStackStaticRouteRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the virtual machine details
	staticroute, count, err := cs.VPC.GetStaticRouteByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Static route %s does no longer exist", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("cidr", staticroute.Cidr)
	d.Set("vpc_id", staticroute.Vpcid)

	return nil
}

func resourceCloudStackStaticRouteDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VPC.NewDeleteStaticRouteParams(d.Id())

	// Delete the private gateway
	_, err := cs.VPC.DeleteStaticRoute(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting static route for %s: %s", d.Get("cidr").(string), err)
	}
	return nil
}

func verifyStaticRouteParams(d *schema.ResourceData) error {
	_, gateway := d.GetOk("gateway_id")
	_, vpc := d.GetOk("vpc_id")
	_, nexthop := d.GetOk("nexthop")

	if (gateway && vpc) || (!gateway && !vpc) {
		return fmt.Errorf(
			"You must supply a value for either (so not both) the 'gateway_id' or 'vpc_id' parameter")
	}

	if (vpc && !nexthop) || (!vpc && nexthop) {
		return fmt.Errorf(
			"Nexthop is required if vpc_id is provided and is not allowed when gateway_id is provided")
	}

	return nil
}
