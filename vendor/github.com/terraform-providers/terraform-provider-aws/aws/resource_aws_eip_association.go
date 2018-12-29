package aws

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/aws/aws-sdk-go/aws"
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

	var resp *ec2.AssociateAddressOutput
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		var err error
		resp, err = conn.AssociateAddress(request)
		if err != nil {
			if isAWSErr(err, "InvalidInstanceID", "pending instance") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error associating EIP: %s", err)
	}

	log.Printf("[DEBUG] EIP Assoc Response: %s", resp)

	supportedPlatforms := meta.(*AWSClient).supportedplatforms
	if len(supportedPlatforms) > 0 && !hasEc2Classic(supportedPlatforms) && resp.AssociationId == nil {
		// We expect no association ID in EC2 Classic
		// but still error out if ID is missing and we _know_ it's NOT EC2 Classic
		return fmt.Errorf("Received no EIP Association ID in account that doesn't support EC2 Classic (%q): %s",
			supportedPlatforms, resp)
	}

	if resp.AssociationId == nil {
		// This is required field for EC2 Classic per docs
		d.SetId(*request.PublicIp)
	} else {
		d.SetId(*resp.AssociationId)
	}

	return resourceAwsEipAssociationRead(d, meta)
}

func resourceAwsEipAssociationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	request, err := describeAddressesById(d.Id(), meta.(*AWSClient).supportedplatforms)
	if err != nil {
		return err
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

	var opts *ec2.DisassociateAddressInput
	// We assume EC2 Classic if ID is a valid IPv4 address
	ip := net.ParseIP(d.Id())
	if ip != nil {
		supportedPlatforms := meta.(*AWSClient).supportedplatforms
		if len(supportedPlatforms) > 0 && !hasEc2Classic(supportedPlatforms) {
			return fmt.Errorf("Received IPv4 address as ID in account that doesn't support EC2 Classic (%q)",
				supportedPlatforms)
		}

		opts = &ec2.DisassociateAddressInput{
			PublicIp: aws.String(d.Id()),
		}
	} else {
		opts = &ec2.DisassociateAddressInput{
			AssociationId: aws.String(d.Id()),
		}
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

func describeAddressesById(id string, supportedPlatforms []string) (*ec2.DescribeAddressesInput, error) {
	// We assume EC2 Classic if ID is a valid IPv4 address
	ip := net.ParseIP(id)
	if ip != nil {
		if len(supportedPlatforms) > 0 && !hasEc2Classic(supportedPlatforms) {
			return nil, fmt.Errorf("Received IPv4 address as ID in account that doesn't support EC2 Classic (%q)",
				supportedPlatforms)
		}

		return &ec2.DescribeAddressesInput{
			Filters: []*ec2.Filter{
				&ec2.Filter{
					Name:   aws.String("public-ip"),
					Values: []*string{aws.String(id)},
				},
				&ec2.Filter{
					Name:   aws.String("domain"),
					Values: []*string{aws.String("standard")},
				},
			},
		}, nil
	}

	return &ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("association-id"),
				Values: []*string{aws.String(id)},
			},
		},
	}, nil
}
