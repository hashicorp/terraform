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

func resourceAwsDirectConnectIntraVirtualInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDirectConnectIntraVirtualInterfaceCreate,
		Read:   resourceAwsDirectConnectIntraVirtualInterfaceRead,
		Delete: resourceAwsDirectConnectIntraVirtualInterfaceDelete,

		Schema: map[string]*schema.Schema{
			"connection_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"owner_account_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"interface_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

func resourceAwsDirectConnectIntraVirtualInterfaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dcconn

	var err error
	var resp *directconnect.VirtualInterface

	if v, ok := d.GetOk("interface_type"); ok && v == "public" {

		createOpts := &directconnect.AllocatePublicVirtualInterfaceInput{
			ConnectionId: aws.String(d.Get("connection_id").(string)),
			NewPublicVirtualInterfaceAllocation: &directconnect.NewPublicVirtualInterfaceAllocation{
				Asn:                  aws.Int64(int64(d.Get("asn").(int))),
				VirtualInterfaceName: aws.String(d.Get("virtual_interface_name").(string)),
				Vlan:                 aws.Int64(int64(d.Get("vlan").(int))),
				RouteFilterPrefixes:  []*directconnect.RouteFilterPrefix{},
			},
			OwnerAccount: aws.String(d.Get("owner_account_id").(string)),
		}

		if v, ok := d.GetOk("amazon_address"); ok {
			createOpts.NewPublicVirtualInterfaceAllocation.AmazonAddress = aws.String(v.(string))
		}

		if v, ok := d.GetOk("auth_key"); ok {
			createOpts.NewPublicVirtualInterfaceAllocation.AuthKey = aws.String(v.(string))
		}

		if v, ok := d.GetOk("customer_address"); ok {
			createOpts.NewPublicVirtualInterfaceAllocation.CustomerAddress = aws.String(v.(string))
		}

		if prefixesSet, ok := d.Get("route_filter_prefixes").(*schema.Set); ok {

			for _, cidr := range prefixesSet.List() {
				createOpts.NewPublicVirtualInterfaceAllocation.RouteFilterPrefixes = append(createOpts.NewPublicVirtualInterfaceAllocation.RouteFilterPrefixes, &directconnect.RouteFilterPrefix{Cidr: aws.String(cidr.(string))})
			}

		}

		log.Printf("[DEBUG] Creating DirectConnect public virtual interface")
		resp, err = conn.AllocatePublicVirtualInterface(createOpts)
		if err != nil {
			return fmt.Errorf("Error creating DirectConnect Virtual Interface: %s", err)
		}

	} else {

		createOpts := &directconnect.AllocatePrivateVirtualInterfaceInput{
			ConnectionId: aws.String(d.Get("connection_id").(string)),
			NewPrivateVirtualInterfaceAllocation: &directconnect.NewPrivateVirtualInterfaceAllocation{
				Asn:                  aws.Int64(int64(d.Get("asn").(int))),
				VirtualInterfaceName: aws.String(d.Get("virtual_interface_name").(string)),
				Vlan:                 aws.Int64(int64(d.Get("vlan").(int))),
			},
			OwnerAccount: aws.String(d.Get("owner_account_id").(string)),
		}

		if v, ok := d.GetOk("amazon_address"); ok {
			createOpts.NewPrivateVirtualInterfaceAllocation.AmazonAddress = aws.String(v.(string))
		}

		if v, ok := d.GetOk("auth_key"); ok {
			createOpts.NewPrivateVirtualInterfaceAllocation.AuthKey = aws.String(v.(string))
		}

		if v, ok := d.GetOk("customer_address"); ok {
			createOpts.NewPrivateVirtualInterfaceAllocation.CustomerAddress = aws.String(v.(string))
		}

		log.Printf("[DEBUG] Creating DirectConnect private virtual interface")
		resp, err = conn.AllocatePrivateVirtualInterface(createOpts)
		if err != nil {
			return fmt.Errorf("Error creating DirectConnect Virtual Interface: %s", err)
		}

	}

	// Store the ID
	VirtualInterface := resp
	d.SetId(*VirtualInterface.VirtualInterfaceId)
	log.Printf("[INFO] VirtualInterface ID: %s", *VirtualInterface.VirtualInterfaceId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"available", "confirming", "verifying", "pending"},
		Refresh:    DirectConnectIntraVirtualInterfaceRefreshFunc(conn, *VirtualInterface.VirtualInterfaceId),
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
	return resourceAwsDirectConnectIntraVirtualInterfaceRead(d, meta)
}

func DirectConnectIntraVirtualInterfaceRefreshFunc(conn *directconnect.DirectConnect, virtualinterfaceId string) resource.StateRefreshFunc {
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

func resourceAwsDirectConnectIntraVirtualInterfaceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dcconn

	resp, err := conn.DescribeVirtualInterfaces(&directconnect.DescribeVirtualInterfacesInput{
		VirtualInterfaceId: aws.String(d.Id()),
	})

	if err != nil {

		log.Printf("[ERROR] Error finding DirectConnect PrivateVirtualInterface: %s", err)
		return err

	}

	if len(resp.VirtualInterfaces) != 1 {
		return fmt.Errorf("[ERROR] Error finding DirectConnect PrivateVirtualInterface: %s", d.Id())
	}

	virtualInterface := resp.VirtualInterfaces[0]

	// Set attributes under the user's control.
	d.Set("connection_id", *virtualInterface.ConnectionId)
	d.Set("asn", *virtualInterface.Asn)
	d.Set("virtual_interface_name", *virtualInterface.VirtualInterfaceName)
	d.Set("vlan", *virtualInterface.Vlan)
	d.Set("amazon_address", *virtualInterface.AmazonAddress)
	d.Set("customer_address", *virtualInterface.CustomerAddress)
	// d.Set("auth_key", *virtualInterface.AuthKey)

	// Set read only attributes.
	d.SetId(*virtualInterface.VirtualInterfaceId)
	d.Set("owner_account_id", *virtualInterface.OwnerAccount)

	return nil
}

func resourceAwsDirectConnectIntraVirtualInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dcconn

	_, err := conn.DeleteVirtualInterface(&directconnect.DeleteVirtualInterfaceInput{
		VirtualInterfaceId: aws.String(d.Id()),
	})

	if err != nil {

		log.Printf("[ERROR] Error deleting DirectConnect PrivateVirtualInterface connection: %s", err)
		return err

	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{"deleted"},
		Refresh:    DirectConnectIntraVirtualInterfaceRefreshFunc(conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for DirectConnect PrivateVirtualInterface (%s) to delete: %s", d.Id(), err)
	}

	return nil
}
