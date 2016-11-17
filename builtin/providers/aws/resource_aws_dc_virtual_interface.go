package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDirectConnectVirtualInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDirectConnectVirtualInterfaceCreate,
		Read:   resourceAwsDirectConnectVirtualInterfaceRead,
		Delete: resourceAwsDirectConnectVirtualInterfaceDelete,

		Schema: map[string]*schema.Schema{
			"connection_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"asn": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"virtual_interface_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"virtual_gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"vlan": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"amazon_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"customer_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"auth_key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"interface_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"route_filter_prefixes": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceAwsDirectConnectVirtualInterfaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dcconn

	var err error
	var resp *directconnect.VirtualInterface

	if v, ok := d.GetOk("interface_type"); ok && v.(string) == "public" {

		createOpts := &directconnect.CreatePublicVirtualInterfaceInput{
			ConnectionId: aws.String(d.Get("connection_id").(string)),
			NewPublicVirtualInterface: &directconnect.NewPublicVirtualInterface{
				Asn:                  aws.Int64(int64(d.Get("asn").(int))),
				VirtualInterfaceName: aws.String(d.Get("virtual_interface_name").(string)),
				Vlan:                 aws.Int64(int64(d.Get("vlan").(int))),
			},
		}

		if v, ok := d.GetOk("amazon_address"); ok {
			createOpts.NewPublicVirtualInterface.AmazonAddress = aws.String(v.(string))
		}

		if v, ok := d.GetOk("auth_key"); ok {
			createOpts.NewPublicVirtualInterface.AuthKey = aws.String(v.(string))
		}

		if v, ok := d.GetOk("customer_address"); ok {
			createOpts.NewPublicVirtualInterface.CustomerAddress = aws.String(v.(string))
		}

		if prefixesSet, ok := d.Get("route_filter_prefixes").(*schema.Set); ok {

			createOpts.NewPublicVirtualInterface.RouteFilterPrefixes = []*directconnect.RouteFilterPrefix{}

			for _, cidr := range prefixesSet.List() {
				createOpts.NewPublicVirtualInterface.RouteFilterPrefixes = append(createOpts.NewPublicVirtualInterface.RouteFilterPrefixes, &directconnect.RouteFilterPrefix{Cidr: aws.String(cidr.(string))})
			}

		}

		log.Println("[DEBUG] request structure: ", createOpts)
		// Create the DirectConnect Connection
		log.Printf("[DEBUG] Creating DirectConnect public virtual interface")
		resp, err = conn.CreatePublicVirtualInterface(createOpts)
		if err != nil {
			return fmt.Errorf("Error creating DirectConnect public virtual interface: %s", err)
		}

	} else {

		createOpts := &directconnect.CreatePrivateVirtualInterfaceInput{
			ConnectionId: aws.String(d.Get("connection_id").(string)),
			NewPrivateVirtualInterface: &directconnect.NewPrivateVirtualInterface{
				Asn:                  aws.Int64(int64(d.Get("asn").(int))),
				VirtualGatewayId:     aws.String(d.Get("virtual_gateway_id").(string)),
				VirtualInterfaceName: aws.String(d.Get("virtual_interface_name").(string)),
				Vlan:                 aws.Int64(int64(d.Get("vlan").(int))),
			},
		}

		if v, ok := d.GetOk("amazon_address"); ok {
			createOpts.NewPrivateVirtualInterface.AmazonAddress = aws.String(v.(string))
		}

		if v, ok := d.GetOk("auth_key"); ok {
			createOpts.NewPrivateVirtualInterface.AuthKey = aws.String(v.(string))
		}

		if v, ok := d.GetOk("customer_address"); ok {
			createOpts.NewPrivateVirtualInterface.CustomerAddress = aws.String(v.(string))
		}

		log.Println("[DEBUG] request structure: ", createOpts)
		// Create the DirectConnect Connection
		log.Printf("[DEBUG] Creating DirectConnect private virtual interface")
		resp, err = conn.CreatePrivateVirtualInterface(createOpts)
		if err != nil {
			return fmt.Errorf("Error creating DirectConnect connection: %s", err)
		}

	}

	// Store the ID
	VirtualInterface := resp
	d.SetId(*VirtualInterface.VirtualInterfaceId)
	log.Printf("[INFO] PrivateVirtualInterface ID: %s", *VirtualInterface.VirtualInterfaceId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending", "down"},
		Target:     []string{"available", "confirming", "verifying"},
		Refresh:    DirectConnectVirtualInterfaceRefreshFunc(conn, *VirtualInterface.VirtualInterfaceId),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for DirectConnect PrivateVirtualInterface (%s) to become ready: %s",
			*VirtualInterface.VirtualInterfaceId, err)
	}

	// Read off the API to populate our RO fields.
	return resourceAwsDirectConnectVirtualInterfaceRead(d, meta)
}

func DirectConnectVirtualInterfaceRefreshFunc(conn *directconnect.DirectConnect, virtualinterfaceId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {

		resp, err := conn.DescribeVirtualInterfaces(&directconnect.DescribeVirtualInterfacesInput{
			VirtualInterfaceId: aws.String(virtualinterfaceId),
		})

		if err != nil {

			log.Printf("Error on DirectConnectPrivateVirtualInterfaceRefresh: %s", err)
			return nil, "", err

		}

		if resp == nil || len(resp.VirtualInterfaces) == 0 {
			return nil, "", nil
		}

		virtualInterface := resp.VirtualInterfaces[0]
		return virtualInterface, *virtualInterface.VirtualInterfaceState, nil
	}
}

func resourceAwsDirectConnectVirtualInterfaceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dcconn

	resp, err := conn.DescribeVirtualInterfaces(&directconnect.DescribeVirtualInterfacesInput{
		VirtualInterfaceId: aws.String(d.Id()),
	})

	if err != nil {

		log.Printf("[ERROR] Error finding DirectConnect VirtualInterface: %s", err)
		return err

	}

	if len(resp.VirtualInterfaces) != 1 {
		return fmt.Errorf("[ERROR] Error finding DirectConnect VirtualInterface: %s", d.Id())
	}

	virtualInterface := resp.VirtualInterfaces[0]

	// Set attributes under the user's control.
	d.Set("connection_id", *virtualInterface.ConnectionId)
	d.Set("asn", *virtualInterface.Asn)
	d.Set("virtual_interface_name", *virtualInterface.VirtualInterfaceName)

	if v, ok := d.GetOk("interface_type"); !ok || (ok && v.(string) == "private") {
		d.Set("virtual_gateway_id", *virtualInterface.VirtualGatewayId)
	}

	d.Set("vlan", *virtualInterface.Vlan)
	d.Set("amazon_address", *virtualInterface.AmazonAddress)
	d.Set("customer_address", *virtualInterface.CustomerAddress)

	// Set read only attributes.
	d.SetId(*virtualInterface.VirtualInterfaceId)

	return nil
}

func resourceAwsDirectConnectVirtualInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dcconn

	_, err := conn.DeleteVirtualInterface(&directconnect.DeleteVirtualInterfaceInput{
		VirtualInterfaceId: aws.String(d.Id()),
	})

	if err != nil {

		log.Printf("[ERROR] Error deleting DirectConnect VirtualInterface connection: %s", err)
		return err

	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{"deleted"},
		Refresh:    DirectConnectVirtualInterfaceRefreshFunc(conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for DirectConnect VirtualInterface (%s) to delete: %s", d.Id(), err)
	}

	return nil
}
