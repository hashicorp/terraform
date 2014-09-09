package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
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

			"allocation_id": &schema.Schema{
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
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	// By default, we're not in a VPC
	domainOpt := ""
	if v := d.Get("vpc"); v != nil && v.(bool) {
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

	// The domain tells us if we're in a VPC or not
	d.Set("domain", allocResp.Domain)

	// Assign the eips (unique) allocation id for use later
	// the EIP api has a conditional unique ID (really), so
	// if we're in a VPC we need to save the ID as such, otherwise
	// it defaults to using the public IP
	log.Printf("[DEBUG] EIP Allocate: %#v", allocResp)
	if d.Get("domain").(string) == "vpc" {
		d.SetId(allocResp.AllocationId)
	} else {
		d.SetId(allocResp.PublicIp)
	}

	log.Printf("[INFO] EIP ID: %s (domain: %v)", d.Id(), allocResp.Domain)
	return resourceAwsEipUpdate(d, meta)
}

func resourceAwsEipUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	domain := resourceAwsEipDomain(d)

	// Only register with an instance if we have one
	if v, ok := d.GetOk("instance"); ok {
		instanceId := v.(string)

		assocOpts := ec2.AssociateAddress{
			InstanceId: instanceId,
			PublicIp:   d.Id(),
		}

		// more unique ID conditionals
		if domain == "vpc" {
			assocOpts = ec2.AssociateAddress{
				InstanceId:   instanceId,
				AllocationId: d.Id(),
				PublicIp:     "",
			}
		}

		log.Printf("[DEBUG] EIP associate configuration: %#v (domain: %v)", assocOpts, domain)
		_, err := ec2conn.AssociateAddress(&assocOpts)
		if err != nil {
			return fmt.Errorf("Failure associating instances: %s", err)
		}
	}

	return resourceAwsEipRead(d, meta)
}

func resourceAwsEipDelete(d *schema.ResourceData, meta interface{}) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "destroyed",
		Refresh:    resourceEipDeleteRefreshFunc(d, meta),
		Timeout:    3 * time.Minute,
		MinTimeout: 1 * time.Second,
	}
	_, err := stateConf.WaitForState()
	return err
}

func resourceAwsEipRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn

	domain := resourceAwsEipDomain(d)
	id := d.Id()

	assocIds := []string{}
	publicIps := []string{}
	if domain == "vpc" {
		assocIds = []string{id}
	} else {
		publicIps = []string{id}
	}

	log.Printf(
		"[DEBUG] EIP describe configuration: %#v, %#v (domain: %s)",
		assocIds, publicIps, domain)

	describeAddresses, err := ec2conn.Addresses(publicIps, assocIds, nil)
	if err != nil {
		if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidAllocationID.NotFound" {
			d.SetId("")
			return nil
		}

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

func resourceEipDeleteRefreshFunc(
	d *schema.ResourceData,
	meta interface{}) resource.StateRefreshFunc {
	p := meta.(*ResourceProvider)
	ec2conn := p.ec2conn
	domain := resourceAwsEipDomain(d)

	return func() (interface{}, string, error) {
		var err error
		if domain == "vpc" {
			log.Printf(
				"[DEBUG] EIP release (destroy) address allocation: %v",
				d.Id())
			_, err = ec2conn.ReleaseAddress(d.Id())
		} else {
			log.Printf("[DEBUG] EIP release (destroy) address: %v", d.Id())
			_, err = ec2conn.ReleasePublicAddress(d.Id())
		}

		if err == nil {
			return d, "destroyed", nil
		}

		ec2err, ok := err.(*ec2.Error)
		if !ok {
			return d, "error", err
		}
		if ec2err.Code != "InvalidIPAddress.InUse" {
			return d, "error", err
		}

		return d, "pending", nil
	}
}
