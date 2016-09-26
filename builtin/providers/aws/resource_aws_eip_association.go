package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEipAssociation() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEipAssociationCreate,
		Read:   resourceAwsEipAssociationRead,
		Delete: resourceAwsEipAssociationDelete,

		Schema: map[string]*schema.Schema{
			"allocation_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"allow_reassociation": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"instance_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"network_interface_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"private_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsEipAssociationCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.AssociateAddressInput{}

	if v, ok := d.GetOk("allocation_id"); ok {
		request.AllocationId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("allow_reassociation"); ok {
		request.AllowReassociation = aws.Bool(v.(bool))
	}
	if v, ok := d.GetOk("instance_id"); ok {
		request.InstanceId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("network_interface_id"); ok {
		request.NetworkInterfaceId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("private_ip_address"); ok {
		request.PrivateIpAddress = aws.String(v.(string))
	}
	if v, ok := d.GetOk("public_ip"); ok {
		request.PublicIp = aws.String(v.(string))
	}

	log.Printf("[DEBUG] EIP association configuration: %#v", request)

	resp, err := conn.AssociateAddress(request)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return fmt.Errorf("[WARN] Error attaching EIP, message: \"%s\", code: \"%s\"",
				awsErr.Message(), awsErr.Code())
		}
		return err
	}

	d.SetId(*resp.AssociationId)

	return resourceAwsEipAssociationRead(d, meta)
}

func resourceAwsEipAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request := &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("association-id"),
				Values: []*string{aws.String(d.Id())},
			},
		},
	}

	response, err := conn.DescribeAddresses(request)
	if err != nil {
		return fmt.Errorf("Error reading EC2 Elastic IP %s: %#v", d.Get("allocation_id").(string), err)
	}

	if response.Addresses == nil || len(response.Addresses) == 0 {
		log.Printf("[INFO] EIP Association ID Not Found. Refreshing from state")
		d.SetId("")
		return nil
	}

	return readAwsEipAssociation(d, response.Addresses[0])
}

func resourceAwsEipAssociationDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	opts := &ec2.DisassociateAddressInput{
		AssociationId: aws.String(d.Id()),
	}

	_, err := conn.DisassociateAddress(opts)
	if err != nil {
		return fmt.Errorf("Error deleting Elastic IP association: %s", err)
	}

	return nil
}

func readAwsEipAssociation(d *schema.ResourceData, address *ec2.Address) error {
	if err := d.Set("allocation_id", address.AllocationId); err != nil {
		return err
	}
	if err := d.Set("instance_id", address.InstanceId); err != nil {
		return err
	}
	if err := d.Set("network_interface_id", address.NetworkInterfaceId); err != nil {
		return err
	}
	if err := d.Set("private_ip_address", address.PrivateIpAddress); err != nil {
		return err
	}
	if err := d.Set("public_ip", address.PublicIp); err != nil {
		return err
	}

	return nil
}
