package aws

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEip() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEipCreate,
		Read:   resourceAwsEipRead,
		Update: resourceAwsEipUpdate,
		Delete: resourceAwsEipDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"vpc": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"instance": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"network_interface": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"allocation_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"association_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"associate_with_private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsEipCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// By default, we're not in a VPC
	domainOpt := ""
	if v := d.Get("vpc"); v != nil && v.(bool) {
		domainOpt = "vpc"
	}

	allocOpts := &ec2.AllocateAddressInput{
		Domain: aws.String(domainOpt),
	}

	log.Printf("[DEBUG] EIP create configuration: %#v", allocOpts)
	allocResp, err := ec2conn.AllocateAddress(allocOpts)
	if err != nil {
		return fmt.Errorf("Error creating EIP: %s", err)
	}

	// The domain tells us if we're in a VPC or not
	d.Set("domain", allocResp.Domain)

	// Assign the eips (unique) allocation id for use later
	// the EIP api has a conditional unique ID (really), so
	// if we're in a VPC we need to save the ID as such, otherwise
	// it defaults to using the public IP
	log.Printf("[DEBUG] EIP Allocate: %#v", allocResp)
	if d.Get("domain").(string) == "vpc" {
		d.SetId(*allocResp.AllocationId)
	} else {
		d.SetId(*allocResp.PublicIp)
	}

	log.Printf("[INFO] EIP ID: %s (domain: %v)", d.Id(), *allocResp.Domain)
	return resourceAwsEipUpdate(d, meta)
}

func resourceAwsEipRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	domain := resourceAwsEipDomain(d)
	id := d.Id()

	req := &ec2.DescribeAddressesInput{}

	if domain == "vpc" {
		req.AllocationIds = []*string{aws.String(id)}
	} else {
		req.PublicIps = []*string{aws.String(id)}
	}

	log.Printf(
		"[DEBUG] EIP describe configuration: %s (domain: %s)",
		req, domain)

	var err error
	var describeAddresses *ec2.DescribeAddressesOutput

	if d.IsNewResource() {
		err := resource.Retry(15*time.Minute, func() *resource.RetryError {
			describeAddresses, err = ec2conn.DescribeAddresses(req)
			if err != nil {
				awsErr, ok := err.(awserr.Error)
				if ok && (awsErr.Code() == "InvalidAllocationID.NotFound" ||
					awsErr.Code() == "InvalidAddress.NotFound") {
					return resource.RetryableError(err)
				}

				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error retrieving EIP: %s", err)
		}
	} else {
		describeAddresses, err = ec2conn.DescribeAddresses(req)
		if err != nil {
			awsErr, ok := err.(awserr.Error)
			if ok && (awsErr.Code() == "InvalidAllocationID.NotFound" ||
				awsErr.Code() == "InvalidAddress.NotFound") {
				log.Printf("[WARN] EIP not found, removing from state: %s", req)
				return nil
			}
			return err
		}
	}

	// Verify AWS returned our EIP
	if len(describeAddresses.Addresses) != 1 ||
		domain == "vpc" && *describeAddresses.Addresses[0].AllocationId != id ||
		*describeAddresses.Addresses[0].PublicIp != id {
		if err != nil {
			return fmt.Errorf("Unable to find EIP: %#v", describeAddresses.Addresses)
		}
	}

	address := describeAddresses.Addresses[0]

	d.Set("association_id", address.AssociationId)
	if address.InstanceId != nil {
		d.Set("instance", address.InstanceId)
	} else {
		d.Set("instance", "")
	}
	if address.NetworkInterfaceId != nil {
		d.Set("network_interface", address.NetworkInterfaceId)
	} else {
		d.Set("network_interface", "")
	}
	d.Set("private_ip", address.PrivateIpAddress)
	d.Set("public_ip", address.PublicIp)

	// On import (domain never set, which it must've been if we created),
	// set the 'vpc' attribute depending on if we're in a VPC.
	if address.Domain != nil {
		d.Set("vpc", *address.Domain == "vpc")
	}

	d.Set("domain", address.Domain)

	// Force ID to be an Allocation ID if we're on a VPC
	// This allows users to import the EIP based on the IP if they are in a VPC
	if *address.Domain == "vpc" && net.ParseIP(id) != nil {
		log.Printf("[DEBUG] Re-assigning EIP ID (%s) to it's Allocation ID (%s)", d.Id(), *address.AllocationId)
		d.SetId(*address.AllocationId)
	}

	return nil
}

func resourceAwsEipUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	domain := resourceAwsEipDomain(d)

	// Associate to instance or interface if specified
	v_instance, ok_instance := d.GetOk("instance")
	v_interface, ok_interface := d.GetOk("network_interface")

	// If we are updating an EIP that is not newly created, and we are attached to
	// an instance or interface, detach first.
	if (d.Get("instance").(string) != "" || d.Get("association_id").(string) != "") && !d.IsNewResource() {
		if err := disassociateEip(d, meta); err != nil {
			return err
		}
	}

	if ok_instance || ok_interface {
		instanceId := v_instance.(string)
		networkInterfaceId := v_interface.(string)

		assocOpts := &ec2.AssociateAddressInput{
			InstanceId: aws.String(instanceId),
			PublicIp:   aws.String(d.Id()),
		}

		// more unique ID conditionals
		if domain == "vpc" {
			var privateIpAddress *string
			if v := d.Get("associate_with_private_ip").(string); v != "" {
				privateIpAddress = aws.String(v)
			}
			assocOpts = &ec2.AssociateAddressInput{
				NetworkInterfaceId: aws.String(networkInterfaceId),
				InstanceId:         aws.String(instanceId),
				AllocationId:       aws.String(d.Id()),
				PrivateIpAddress:   privateIpAddress,
			}
		}

		log.Printf("[DEBUG] EIP associate configuration: %s (domain: %s)", assocOpts, domain)

		err := resource.Retry(5*time.Minute, func() *resource.RetryError {
			_, err := ec2conn.AssociateAddress(assocOpts)
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					if awsErr.Code() == "InvalidAllocationID.NotFound" {
						return resource.RetryableError(awsErr)
					}
				}
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			// Prevent saving instance if association failed
			// e.g. missing internet gateway in VPC
			d.Set("instance", "")
			d.Set("network_interface", "")
			return fmt.Errorf("Failure associating EIP: %s", err)
		}
	}

	return resourceAwsEipRead(d, meta)
}

func resourceAwsEipDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	if err := resourceAwsEipRead(d, meta); err != nil {
		return err
	}
	if d.Id() == "" {
		// This might happen from the read
		return nil
	}

	// If we are attached to an instance or interface, detach first.
	if d.Get("instance").(string) != "" || d.Get("association_id").(string) != "" {
		if err := disassociateEip(d, meta); err != nil {
			return err
		}
	}

	domain := resourceAwsEipDomain(d)
	return resource.Retry(3*time.Minute, func() *resource.RetryError {
		var err error
		switch domain {
		case "vpc":
			log.Printf(
				"[DEBUG] EIP release (destroy) address allocation: %v",
				d.Id())
			_, err = ec2conn.ReleaseAddress(&ec2.ReleaseAddressInput{
				AllocationId: aws.String(d.Id()),
			})
		case "standard":
			log.Printf("[DEBUG] EIP release (destroy) address: %v", d.Id())
			_, err = ec2conn.ReleaseAddress(&ec2.ReleaseAddressInput{
				PublicIp: aws.String(d.Id()),
			})
		}

		if err == nil {
			return nil
		}
		if _, ok := err.(awserr.Error); !ok {
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(err)
	})
}

func resourceAwsEipDomain(d *schema.ResourceData) string {
	if v, ok := d.GetOk("domain"); ok {
		return v.(string)
	} else if strings.Contains(d.Id(), "eipalloc") {
		// We have to do this for backwards compatibility since TF 0.1
		// didn't have the "domain" computed attribute.
		return "vpc"
	}

	return "standard"
}

func disassociateEip(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	log.Printf("[DEBUG] Disassociating EIP: %s", d.Id())
	var err error
	switch resourceAwsEipDomain(d) {
	case "vpc":
		_, err = ec2conn.DisassociateAddress(&ec2.DisassociateAddressInput{
			AssociationId: aws.String(d.Get("association_id").(string)),
		})
	case "standard":
		_, err = ec2conn.DisassociateAddress(&ec2.DisassociateAddressInput{
			PublicIp: aws.String(d.Get("public_ip").(string)),
		})
	}

	// First check if the association ID is not found. If this
	// is the case, then it was already disassociated somehow,
	// and that is okay. The most commmon reason for this is that
	// the instance or ENI it was attached it was destroyed.
	if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidAssociationID.NotFound" {
		err = nil
	}
	return err
}
