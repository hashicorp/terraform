package aws

import (
	"fmt"
	"log"

	//"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
)

func resourceAwsDhcpOptionSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDhcpOptionSetCreate,
		Delete: resourceAwsDhcpOptionSetDelete,

		Schema: map[string]*schema.Schema{
			"domain_name_servers": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ntp_servers": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"netbios_name_servers": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"netbios_node_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsDhcpOptionSetCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// Create DHCP Options set options
	createDhcpOpts := &ec2.CreateDhcpOptions{
		DomainNameServers:  d.Get("domain_name_servers").(string),
		DomainName:         d.Get("domain_name").(string),
		NtpServers:         d.Get("ntp_servers").(string),
		NetbiosNameServers: d.Get("netbios_name_servers").(string),
		NetbiosNodeType:    d.Get("netbios_node_type").(string),
	}

	// Create the DHCP Options set
	log.Printf("[DEBUG] DHCP Options create config: %#v", createDhcpOpts)
	dhcpResp, err := ec2conn.CreateDhcpOptions(createDhcpOpts)
	if err != nil {
		return fmt.Errorf("Error creating DHCP Options: %s", err)
	}

	// Get the ID
	dhcp := &dhcpResp.DhcpOptions
	log.Printf("[INFO] DHCP Options Set ID: %s", dhcp.DhcpOptionsId)
	d.SetId(dhcp.DhcpOptionsId)

	return nil
}

func resourceAwsDhcpOptionSetDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Deleting DHCP Options Set: %s", d.Id())

	if _, err := ec2conn.DeleteDhcpOptions(d.Id()); err != nil {
		return fmt.Errorf("Error deleting DHCP Options Set: %s", err)
	}

	return nil
}
