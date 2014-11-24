package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
)

func resourceAwsRouteTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRouteTableCreate,
		Read:   resourceAwsRouteTableRead,
		Update: resourceAwsRouteTableUpdate,
		Delete: resourceAwsRouteTableDelete,

		Schema: map[string]*schema.Schema{
			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"route": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr_block": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"gateway_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"instance_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAwsRouteTableHash,
			},
		},
	}
}

func resourceAwsRouteTableCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// Create the routing table
	createOpts := &ec2.CreateRouteTable{
		VpcId: d.Get("vpc_id").(string),
	}
	log.Printf("[DEBUG] RouteTable create config: %#v", createOpts)

	resp, err := ec2conn.CreateRouteTable(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating route table: %s", err)
	}

	// Get the ID and store it
	rt := &resp.RouteTable
	d.SetId(rt.RouteTableId)
	log.Printf("[INFO] Route Table ID: %s", d.Id())

	// Wait for the route table to become available
	log.Printf(
		"[DEBUG] Waiting for route table (%s) to become available",
		d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "ready",
		Refresh: resourceAwsRouteTableStateRefreshFunc(ec2conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for route table (%s) to become available: %s",
			d.Id(), err)
	}

	return resourceAwsRouteTableUpdate(d, meta)
}

func resourceAwsRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(ec2conn, d.Id())()
	if err != nil {
		return err
	}
	if rtRaw == nil {
		return nil
	}

	rt := rtRaw.(*ec2.RouteTable)
	d.Set("vpc_id", rt.VpcId)

	// Create an empty schema.Set to hold all routes
	route := &schema.Set{F: resourceAwsRouteTableHash}

	// Loop through the routes and add them to the set
	for _, r := range rt.Routes {
		if r.GatewayId == "local" {
			continue
		}

		m := make(map[string]interface{})
		m["cidr_block"] = r.DestinationCidrBlock

		if r.GatewayId != "" {
			m["gateway_id"] = r.GatewayId
		}

		if r.InstanceId != "" {
			m["instance_id"] = r.InstanceId
		}

		route.Add(m)
	}
	d.Set("route", route)

	return nil
}

func resourceAwsRouteTableUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// Check if the route set as a whole has changed
	if d.HasChange("route") {
		o, n := d.GetChange("route")
		ors := o.(*schema.Set).Difference(n.(*schema.Set))
		nrs := n.(*schema.Set).Difference(o.(*schema.Set))

		// Now first loop through all the old routes and delete any obsolete ones
		for _, route := range ors.List() {
			m := route.(map[string]interface{})

			// Delete the route as it no longer exists in the config
			_, err := ec2conn.DeleteRoute(
				d.Id(), m["cidr_block"].(string))
			if err != nil {
				return err
			}
		}

		// Make sure we save the state of the currently configured rules
		routes := o.(*schema.Set).Intersection(n.(*schema.Set))
		d.Set("route", routes)

		// Then loop through al the newly configured routes and create them
		for _, route := range nrs.List() {
			m := route.(map[string]interface{})

			opts := ec2.CreateRoute{
				RouteTableId:         d.Id(),
				DestinationCidrBlock: m["cidr_block"].(string),
				GatewayId:            m["gateway_id"].(string),
				InstanceId:           m["instance_id"].(string),
			}

			_, err := ec2conn.CreateRoute(&opts)
			if err != nil {
				return err
			}

			routes.Add(route)
			d.Set("route", routes)
		}
	}

	return resourceAwsRouteTableRead(d, meta)
}

func resourceAwsRouteTableDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// First request the routing table since we'll have to disassociate
	// all the subnets first.
	rtRaw, _, err := resourceAwsRouteTableStateRefreshFunc(ec2conn, d.Id())()
	if err != nil {
		return err
	}
	if rtRaw == nil {
		return nil
	}
	rt := rtRaw.(*ec2.RouteTable)

	// Do all the disassociations
	for _, a := range rt.Associations {
		log.Printf("[INFO] Disassociating association: %s", a.AssociationId)
		if _, err := ec2conn.DisassociateRouteTable(a.AssociationId); err != nil {
			return err
		}
	}

	// Delete the route table
	log.Printf("[INFO] Deleting Route Table: %s", d.Id())
	if _, err := ec2conn.DeleteRouteTable(d.Id()); err != nil {
		ec2err, ok := err.(*ec2.Error)
		if ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
			return nil
		}

		return fmt.Errorf("Error deleting route table: %s", err)
	}

	// Wait for the route table to really destroy
	log.Printf(
		"[DEBUG] Waiting for route table (%s) to become destroyed",
		d.Id())

	stateConf := &resource.StateChangeConf{
		Pending: []string{"ready"},
		Target:  "",
		Refresh: resourceAwsRouteTableStateRefreshFunc(ec2conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for route table (%s) to become destroyed: %s",
			d.Id(), err)
	}

	return nil
}

func resourceAwsRouteTableHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["cidr_block"].(string)))

	if v, ok := m["gateway_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["instance_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

// resourceAwsRouteTableStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a RouteTable.
func resourceAwsRouteTableStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeRouteTables([]string{id}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidRouteTableID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on RouteTableStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		rt := &resp.RouteTables[0]
		return rt, "ready", nil
	}
}
