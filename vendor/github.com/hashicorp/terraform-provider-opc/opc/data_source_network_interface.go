package opc

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-oracle-terraform/compute"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNetworkInterface() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkInterfaceRead,

		Schema: map[string]*schema.Schema{
			"instance_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			"instance_name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"interface": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Computed Values returned from the data source lookup
			"dns": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_network": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"model": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"nat": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"search_domains": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"sec_lists": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"shared_network": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"vnic": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vnic_sets": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceNetworkInterfaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*compute.Client).Instances()

	// Get required attributes
	instance_name := d.Get("instance_name").(string)
	instance_id := d.Get("instance_id").(string)
	targetInterface := d.Get("interface").(string)

	// Get instance
	input := &compute.GetInstanceInput{
		Name: instance_name,
		ID:   instance_id,
	}

	instance, err := client.GetInstance(input)
	if err != nil {
		if compute.WasNotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading instance %q: %v", instance_name, err)
	}

	result := compute.NetworkingInfo{}

	// If the target instance has no network interfaces, return
	if instance.Networking == nil {
		d.SetId("")
		return nil
	}

	// Set the computed fields
	result = instance.Networking[targetInterface]

	// Check if the target interface exists or not
	if &result == nil {
		log.Printf("[WARN] %q networking interface not found on instance %q", targetInterface, instance_name)
	}

	d.SetId(fmt.Sprintf("%s-%s", instance_name, targetInterface))

	// vNIC is a required field for an IP Network interface, and can only be set if the network
	// interface is inside an IP Network. Use this key to determine shared_network status
	if result.Vnic != "" {
		d.Set("shared_network", false)
	} else {
		d.Set("shared_network", true)
	}

	d.Set("ip_address", result.IPAddress)
	d.Set("ip_network", result.IPNetwork)
	d.Set("mac_address", result.MACAddress)
	d.Set("model", result.Model)
	d.Set("vnic", result.Vnic)

	if err := setStringList(d, "dns", result.DNS); err != nil {
		return err
	}
	if err := setStringList(d, "name_servers", result.NameServers); err != nil {
		return err
	}
	if err := setStringList(d, "nat", result.Nat); err != nil {
		return err
	}
	if err := setStringList(d, "search_domains", result.SearchDomains); err != nil {
		return err
	}
	if err := setStringList(d, "sec_lists", result.SecLists); err != nil {
		return err
	}
	if err := setStringList(d, "vnic_sets", result.VnicSets); err != nil {
		return err
	}

	return nil
}
