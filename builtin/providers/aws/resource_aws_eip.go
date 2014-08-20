package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	//"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/goamz/ec2"
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
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// By default, we're not in a VPC
	vpc := false
	domainOpt := ""
	if v := d.Get("vpc"); v != nil && v.(bool) {
		vpc = true
		domainOpt = "vpc"
	}

	allocOpts := ec2.AllocateAddress{
		Domain: domainOpt,
	}

	log.Printf("[DEBUG] EIP create configuration: %#v", allocOpts)
	allocResp, err := ec2conn.AllocateAddress(&allocOpts)
	if err != nil {
		return fmt.Errorf("Error creating EIP: %s", err)
	}

	// Assign the eips (unique) allocation id for use later
	// the EIP api has a conditional unique ID (really), so
	// if we're in a VPC we need to save the ID as such, otherwise
	// it defaults to using the public IP
	log.Printf("[DEBUG] EIP Allocate: %#v", allocResp)
	if allocResp.AllocationId != "" {
		d.SetId(allocResp.AllocationId)
		d.Set("vpc", true)
	} else {
		d.SetId(allocResp.PublicIp)
		d.Set("vpc", false)
	}

	log.Printf("[INFO] EIP ID: %s (vpc: %v)", d.Id(), vpc)
	return resourceAwsEipRead(d, meta)
}

func resourceAwsEipUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	vpc := strings.Contains(d.Id(), "eipalloc")

	// Only register with an instance if we have one
	if v := d.Get("instance"); v != nil {
		instanceId := v.(string)

		assocOpts := ec2.AssociateAddress{
			InstanceId: instanceId,
			PublicIp:   d.Id(),
		}

		// more unique ID conditionals
		if vpc {
			assocOpts = ec2.AssociateAddress{
				InstanceId:   instanceId,
				AllocationId: d.Id(),
				PublicIp:     "",
			}
		}

		log.Printf("[DEBUG] EIP associate configuration: %#v (vpc: %v)", assocOpts, vpc)
		_, err := ec2conn.AssociateAddress(&assocOpts)
		if err != nil {
			return fmt.Errorf("Failure associating instances: %s", err)
		}
	}

	return resourceAwsEipRead(d, meta)
}

func resourceAwsEipDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	var err error
	if strings.Contains(d.Id(), "eipalloc") {
		log.Printf("[DEBUG] EIP release (destroy) address allocation: %v", d.Id())
		_, err = ec2conn.ReleaseAddress(d.Id())
		return err
	} else {
		log.Printf("[DEBUG] EIP release (destroy) address: %v", d.Id())
		_, err = ec2conn.ReleasePublicAddress(d.Id())
		return err
	}

	return nil
}

func resourceAwsEipRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	vpc := false
	if d.Get("vpc").(bool) {
		vpc = true
	}

	id := d.Id()

	assocIds := []string{}
	publicIps := []string{}
	if vpc {
		assocIds = []string{id}
	} else {
		publicIps = []string{id}
	}

	log.Printf(
		"[DEBUG] EIP describe configuration: %#v, %#v (vpc: %v)",
		assocIds, publicIps, vpc)

	describeAddresses, err := ec2conn.Addresses(publicIps, assocIds, nil)
	if err != nil {
		return fmt.Errorf("Error retrieving EIP: %s", err)
	}

	// Verify AWS returned our EIP
	if len(describeAddresses.Addresses) != 1 ||
		describeAddresses.Addresses[0].AllocationId != id ||
		describeAddresses.Addresses[0].PublicIp != id {
		if err != nil {
			return fmt.Errorf("Unable to find EIP: %#v", describeAddresses.Addresses)
		}
	}

	address := describeAddresses.Addresses[0]

	d.Set("instance", address.InstanceId)
	d.Set("public_ip", address.PublicIp)
	d.Set("private_ip", address.PrivateIpAddress)

	return nil
}
