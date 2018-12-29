package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/directconnect"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsDxHostedPublicVirtualInterface() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDxHostedPublicVirtualInterfaceCreate,
		Read:   resourceAwsDxHostedPublicVirtualInterfaceRead,
		Delete: resourceAwsDxHostedPublicVirtualInterfaceDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsDxHostedPublicVirtualInterfaceImport,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"connection_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"vlan": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntBetween(1, 4094),
			},
			"bgp_asn": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"bgp_auth_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"address_family": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{directconnect.AddressFamilyIpv4, directconnect.AddressFamilyIpv6}, false),
			},
			"customer_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"amazon_address": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"owner_account_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsAccountId,
			},
			"route_filter_prefixes": {
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				MinItems: 1,
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},
	}
}

func resourceAwsDxHostedPublicVirtualInterfaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dxconn

	addressFamily := d.Get("address_family").(string)
	caRaw, caOk := d.GetOk("customer_address")
	aaRaw, aaOk := d.GetOk("amazon_address")
	if addressFamily == directconnect.AddressFamilyIpv4 {
		if !caOk {
			return fmt.Errorf("'customer_address' must be set when 'address_family' is '%s'", addressFamily)
		}
		if !aaOk {
			return fmt.Errorf("'amazon_address' must be set when 'address_family' is '%s'", addressFamily)
		}
	}

	req := &directconnect.AllocatePublicVirtualInterfaceInput{
		ConnectionId: aws.String(d.Get("connection_id").(string)),
		OwnerAccount: aws.String(d.Get("owner_account_id").(string)),
		NewPublicVirtualInterfaceAllocation: &directconnect.NewPublicVirtualInterfaceAllocation{
			VirtualInterfaceName: aws.String(d.Get("name").(string)),
			Vlan:                 aws.Int64(int64(d.Get("vlan").(int))),
			Asn:                  aws.Int64(int64(d.Get("bgp_asn").(int))),
			AddressFamily:        aws.String(addressFamily),
		},
	}
	if v, ok := d.GetOk("bgp_auth_key"); ok && v.(string) != "" {
		req.NewPublicVirtualInterfaceAllocation.AuthKey = aws.String(v.(string))
	}
	if caOk && caRaw.(string) != "" {
		req.NewPublicVirtualInterfaceAllocation.CustomerAddress = aws.String(caRaw.(string))
	}
	if aaOk && aaRaw.(string) != "" {
		req.NewPublicVirtualInterfaceAllocation.AmazonAddress = aws.String(aaRaw.(string))
	}
	if v, ok := d.GetOk("route_filter_prefixes"); ok {
		req.NewPublicVirtualInterfaceAllocation.RouteFilterPrefixes = expandDxRouteFilterPrefixes(v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] Allocating Direct Connect hosted public virtual interface: %#v", req)
	resp, err := conn.AllocatePublicVirtualInterface(req)
	if err != nil {
		return fmt.Errorf("Error allocating Direct Connect hosted public virtual interface: %s", err.Error())
	}

	d.SetId(aws.StringValue(resp.VirtualInterfaceId))
	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Region:    meta.(*AWSClient).region,
		Service:   "directconnect",
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("dxvif/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	if err := dxHostedPublicVirtualInterfaceWaitUntilAvailable(conn, d.Id(), d.Timeout(schema.TimeoutCreate)); err != nil {
		return err
	}

	return resourceAwsDxHostedPublicVirtualInterfaceRead(d, meta)
}

func resourceAwsDxHostedPublicVirtualInterfaceRead(d *schema.ResourceData, meta interface{}) error {
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

	d.Set("connection_id", vif.ConnectionId)
	d.Set("name", vif.VirtualInterfaceName)
	d.Set("vlan", vif.Vlan)
	d.Set("bgp_asn", vif.Asn)
	d.Set("bgp_auth_key", vif.AuthKey)
	d.Set("address_family", vif.AddressFamily)
	d.Set("customer_address", vif.CustomerAddress)
	d.Set("amazon_address", vif.AmazonAddress)
	d.Set("route_filter_prefixes", flattenDxRouteFilterPrefixes(vif.RouteFilterPrefixes))
	d.Set("owner_account_id", vif.OwnerAccount)

	return nil
}

func resourceAwsDxHostedPublicVirtualInterfaceDelete(d *schema.ResourceData, meta interface{}) error {
	return dxVirtualInterfaceDelete(d, meta)
}

func resourceAwsDxHostedPublicVirtualInterfaceImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
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

func dxHostedPublicVirtualInterfaceWaitUntilAvailable(conn *directconnect.DirectConnect, vifId string, timeout time.Duration) error {
	return dxVirtualInterfaceWaitUntilAvailable(
		conn,
		vifId,
		timeout,
		[]string{
			directconnect.VirtualInterfaceStatePending,
		},
		[]string{
			directconnect.VirtualInterfaceStateAvailable,
			directconnect.VirtualInterfaceStateConfirming,
			directconnect.VirtualInterfaceStateDown,
			directconnect.VirtualInterfaceStateVerifying,
		})
}
