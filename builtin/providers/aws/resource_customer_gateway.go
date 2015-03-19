package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
)

func resourceAwsCustomerGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCustomerGatewayCreate,
		Read:   resourceAwsCustomerGatewayRead,
		Update: resourceAwsCustomerGatewayUpdate,
		Delete: resourceAwsCustomerGatewayDelete,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"bgp_asn": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default: 65000,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsCustomerGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	createOpts := &ec2.CreateCustomerGateway{
		Type:            d.Get("type").(string),
		IpAddress:       d.Get("ip_address").(string),
		BgpAsn:          d.Get("bgp_asn").(int),
	}

	// Create the gateway
	log.Printf("[DEBUG] Creating customer gateway")
	resp, err := ec2conn.CreateCustomerGateway(createOpts)

	if err != nil {
		return fmt.Errorf("Error creating customer gateway: %s", err)
	}

	// Get the ID and store it
	customerGateway := &resp.CustomerGateway
	log.Printf("[INFO] CustomerGateway ID: %s", customerGateway.CustomerGatewayId)
	d.SetId(customerGateway.CustomerGatewayId)

	return resourceAwsCustomerGatewayUpdate(d, meta)
}

func resourceAwsCustomerGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	d.Partial(true)

	if err := setTags(ec2conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsCustomerGatewayRead(d, meta)
}

func resourceAwsCustomerGatewayRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	resp, err := ec2conn.DescribeCustomerGateways([]string{d.Id()}, ec2.NewFilter())

	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}

	customerGateway := &resp.CustomerGateways[0]

	d.Set("type", customerGateway.Type)
	d.Set("ip_address", customerGateway.IpAddress)
	d.Set("bgp_asn", customerGateway.BgpAsn)
	d.Set("tags", tagsToMap(customerGateway.Tags))
	return nil
}


func resourceAwsCustomerGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Deleting Customer Gateway: %s", d.Id())

	return resource.Retry(5*time.Minute, func() error {
			_, err := ec2conn.DeleteCustomerGateway(d.Id())
			if err != nil {
				ec2err, ok := err.(*ec2.Error)
				if !ok {
					return err
				}

				switch ec2err.Code {
				case "InvalidCustomerGatewayID.NotFound":
					return nil
				default:
					return resource.RetryError{err}
				}
			}

			log.Printf("[Info] Deleted customer gateway %s successfully", d.Id())
			return nil
	})

}
