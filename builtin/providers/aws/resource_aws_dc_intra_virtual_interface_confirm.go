package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directconnect"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDirectConnectIntraVirtualInterfaceConfirm() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDirectConnectIntraVirtualInterfaceConfirmCreate,
		Read:   resourceAwsDirectConnectIntraVirtualInterfaceConfirmRead,
		Delete: resourceAwsDirectConnectIntraVirtualInterfaceConfirmDelete,

		Schema: map[string]*schema.Schema{

			"interface_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"virtual_interface_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"virtual_gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"allow_down_state": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsDirectConnectIntraVirtualInterfaceConfirmCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dcconn

	var err error

	if v, ok := d.GetOk("interface_type"); ok && v.(string) == "public" {

		createOpts := &directconnect.ConfirmPublicVirtualInterfaceInput{
			VirtualInterfaceId: aws.String(d.Get("virtual_interface_id").(string)),
		}

		log.Printf("[DEBUG] Creating DirectConnect public virtual interface")
		_, err = conn.ConfirmPublicVirtualInterface(createOpts)
		if err != nil {
			return fmt.Errorf("Error creating DirectConnect Virtual Interface: %s", err)
		}

	} else {

		createOpts := &directconnect.ConfirmPrivateVirtualInterfaceInput{
			VirtualInterfaceId: aws.String(d.Get("virtual_interface_id").(string)),
		}

		if v, ok := d.GetOk("virtual_gateway_id"); ok {
			createOpts.VirtualGatewayId = aws.String(v.(string))
		} else {
			return fmt.Errorf("Error virtual_gateway_id is required for private virtual interface with id: %s", d.Get("virtual_interface_id").(string))
		}

		log.Printf("[DEBUG] Creating DirectConnect private virtual interface")
		_, err = conn.ConfirmPrivateVirtualInterface(createOpts)
		if err != nil {
			return fmt.Errorf("Error creating DirectConnect Virtual Interface: %s", err)
		}

	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending", "down"},
		Target:     []string{"available"},
		Refresh:    DirectConnectIntraVirtualInterfaceConfirmRefreshFunc(conn, d.Get("virtual_interface_id").(string)),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 10 * time.Second,
	}

	if v, ok := d.GetOk("allow_down_state"); ok && v.(bool) {
		stateConf.Target = []string{"available", "down"}
		stateConf.Pending = []string{"pending"}
	}

	_, stateErr := stateConf.WaitForState()
	if stateErr != nil {
		return fmt.Errorf(
			"Error waiting for DirectConnect PrivateVirtualInterface (%s) to become ready: %s",
			d.Get("virtual_interface_id").(string), err)
	}

	d.SetId(d.Get("virtual_interface_id").(string))

	// Read off the API to populate our RO fields.
	return resourceAwsDirectConnectIntraVirtualInterfaceConfirmRead(d, meta)
}

func DirectConnectIntraVirtualInterfaceConfirmRefreshFunc(conn *directconnect.DirectConnect, virtualinterfaceId string) resource.StateRefreshFunc {
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

func resourceAwsDirectConnectIntraVirtualInterfaceConfirmRead(d *schema.ResourceData, meta interface{}) error {
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
	d.Set("connection_id", virtualInterface.ConnectionId)
	d.Set("asn", virtualInterface.Asn)
	d.Set("virtual_interface_name", virtualInterface.VirtualInterfaceName)
	d.Set("vlan", virtualInterface.Vlan)
	d.Set("amazon_address", virtualInterface.AmazonAddress)
	d.Set("customer_address", virtualInterface.CustomerAddress)
	// d.Set("auth_key", *virtualInterface.AuthKey)

	// Set read only attributes.
	d.SetId(*virtualInterface.VirtualInterfaceId)
	d.Set("owner_account_id", virtualInterface.OwnerAccount)

	return nil
}

func resourceAwsDirectConnectIntraVirtualInterfaceConfirmDelete(d *schema.ResourceData, meta interface{}) error {
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
