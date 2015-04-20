package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsEip() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEipCreate,
		Read:   resourceAwsEipRead,
		Update: resourceAwsEipUpdate,
		Delete: resourceAwsEipDelete,

		Schema: map[string]*schema.Schema{
			"vpc": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"instance": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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
		d.SetId(*allocResp.AllocationID)
	} else {
		d.SetId(*allocResp.PublicIP)
	}

	log.Printf("[INFO] EIP ID: %s (domain: %v)", d.Id(), *allocResp.Domain)
	return resourceAwsEipUpdate(d, meta)
}

func resourceAwsEipRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	domain := resourceAwsEipDomain(d)
	id := d.Id()

	assocIds := []*string{}
	publicIps := []*string{}
	if domain == "vpc" {
		assocIds = []*string{aws.String(id)}
	} else {
		publicIps = []*string{aws.String(id)}
	}

	log.Printf(
		"[DEBUG] EIP describe configuration: %#v, %#v (domain: %s)",
		assocIds, publicIps, domain)

	req := &ec2.DescribeAddressesInput{
		AllocationIDs: assocIds,
		PublicIPs:     publicIps,
	}
	describeAddresses, err := ec2conn.DescribeAddresses(req)
	if err != nil {
		if ec2err, ok := err.(aws.APIError); ok && ec2err.Code == "InvalidAllocationID.NotFound" {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving EIP: %s", err)
	}

	// Verify AWS returned our EIP
	if len(describeAddresses.Addresses) != 1 ||
		(domain == "vpc" && *describeAddresses.Addresses[0].AllocationID != id) ||
		*describeAddresses.Addresses[0].PublicIP != id {
		if err != nil {
			return fmt.Errorf("Unable to find EIP: %#v", describeAddresses.Addresses)
		}
	}

	address := describeAddresses.Addresses[0]

	d.Set("association_id", address.AssociationID)
	d.Set("instance", address.InstanceID)
	d.Set("private_ip", address.PrivateIPAddress)
	d.Set("public_ip", address.PublicIP)

	return nil
}

func resourceAwsEipUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	domain := resourceAwsEipDomain(d)

	// Only register with an instance if we have one
	if v, ok := d.GetOk("instance"); ok {
		instanceId := v.(string)

		assocOpts := &ec2.AssociateAddressInput{
			InstanceID: aws.String(instanceId),
			PublicIP:   aws.String(d.Id()),
		}

		// more unique ID conditionals
		if domain == "vpc" {
			assocOpts = &ec2.AssociateAddressInput{
				InstanceID:   aws.String(instanceId),
				AllocationID: aws.String(d.Id()),
				PublicIP:     aws.String(""),
			}
		}

		log.Printf("[DEBUG] EIP associate configuration: %#v (domain: %v)", assocOpts, domain)
		_, err := ec2conn.AssociateAddress(assocOpts)
		if err != nil {
			return fmt.Errorf("Failure associating instances: %s", err)
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

	// If we are attached to an instance, detach first.
	if d.Get("instance").(string) != "" {
		log.Printf("[DEBUG] Disassociating EIP: %s", d.Id())
		var err error
		switch resourceAwsEipDomain(d) {
		case "vpc":
			_, err = ec2conn.DisassociateAddress(&ec2.DisassociateAddressInput{
				AssociationID: aws.String(d.Get("association_id").(string)),
			})
		case "standard":
			_, err = ec2conn.DisassociateAddress(&ec2.DisassociateAddressInput{
				PublicIP: aws.String(d.Get("public_ip").(string)),
			})
		}
		if err != nil {
			return err
		}
	}

	domain := resourceAwsEipDomain(d)
	return resource.Retry(3*time.Minute, func() error {
		var err error
		switch domain {
		case "vpc":
			log.Printf(
				"[DEBUG] EIP release (destroy) address allocation: %v",
				d.Id())
			_, err = ec2conn.ReleaseAddress(&ec2.ReleaseAddressInput{
				AllocationID: aws.String(d.Id()),
			})
		case "standard":
			log.Printf("[DEBUG] EIP release (destroy) address: %v", d.Id())
			_, err = ec2conn.ReleaseAddress(&ec2.ReleaseAddressInput{
				PublicIP: aws.String(d.Id()),
			})
		}

		if err == nil {
			return nil
		}
		if _, ok := err.(aws.APIError); !ok {
			return resource.RetryError{Err: err}
		}

		return err
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
