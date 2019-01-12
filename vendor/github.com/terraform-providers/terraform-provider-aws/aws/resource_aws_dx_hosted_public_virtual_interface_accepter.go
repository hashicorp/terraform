package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDxHostedPublicVirtualInterfaceAccepter() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxHostedPublicVirtualInterfaceAccepterCreate,
		Read:   resourceAwsDxHostedPublicVirtualInterfaceAccepterRead,
		Update: resourceAwsDxHostedPublicVirtualInterfaceAccepterUpdate,
		Delete: resourceAwsDxHostedPublicVirtualInterfaceAccepterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsDxHostedPublicVirtualInterfaceAccepterImport,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"virtual_interface_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceAwsDxHostedPublicVirtualInterfaceAccepterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	vifId := d.Get("virtual_interface_id").(string)
	req := &directconnect.ConfirmPublicVirtualInterfaceInput{
		VirtualInterfaceId: aws.String(vifId),
	}

	log.Printf("[DEBUG] Accepting Direct Connect hosted public virtual interface: %#v", req)
	_, err := conn.ConfirmPublicVirtualInterface(req)
	if err != nil {
		return fmt.Errorf("Error accepting Direct Connect hosted public virtual interface: %s", err.Error())
	}

	d.SetId(vifId)
	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxvif/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	if err := dxHostedPublicVirtualInterfaceAccepterWaitUntilAvailable(conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return err
	}

	return resourceAwsDxHostedPublicVirtualInterfaceAccepterUpdate(d, meta)
}

func resourceAwsDxHostedPublicVirtualInterfaceAccepterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	vif, err := dxVirtualInterfaceRead(d.Id(), conn)
	if err != nil {
		return err
	}
	if vif == nil {
		log.Printf("[WARN] Direct Connect virtual interface (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	vifState := aws.StringValue(vif.VirtualInterfaceState)
	if vifState != directconnect.VirtualInterfaceStateAvailable &&
		vifState != directconnect.VirtualInterfaceStateDown &&
		vifState != directconnect.VirtualInterfaceStateVerifying {
		log.Printf("[WARN] Direct Connect virtual interface (%s) is '%s', removing from state", vifState, d.Id())
		d.SetId("")
		return nil
	}

	d.Set("virtual_interface_id", vif.VirtualInterfaceId)
	if err := getTagsDX(conn, d, d.Get("arn").(string)); err != nil {
		return err
	}

	return nil
}

func resourceAwsDxHostedPublicVirtualInterfaceAccepterUpdate(d *schema.ResourceData, meta interface{}) error {
	if err := dxVirtualInterfaceUpdate(d, meta); err != nil {
		return err
	}

	return resourceAwsDxHostedPublicVirtualInterfaceAccepterRead(d, meta)
}

func resourceAwsDxHostedPublicVirtualInterfaceAccepterDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[WARN] Will not delete Direct Connect virtual interface. Terraform will remove this resource from the state file, however resources may remain.")
	return nil
}

func resourceAwsDxHostedPublicVirtualInterfaceAccepterImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxvif/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	return []*schema.ResourceData{d}, nil
}

func dxHostedPublicVirtualInterfaceAccepterWaitUntilAvailable(conn *directconnect.DirectConnect, vifId string, timeout time.Duration) error {
	return dxVirtualInterfaceWaitUntilAvailable(
		conn,
		vifId,
		timeout,
		[]string{
			directconnect.VirtualInterfaceStateConfirming,
			directconnect.VirtualInterfaceStatePending,
		},
		[]string{
			directconnect.VirtualInterfaceStateAvailable,
			directconnect.VirtualInterfaceStateDown,
			directconnect.VirtualInterfaceStateVerifying,
		})
}
