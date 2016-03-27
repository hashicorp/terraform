package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulNode() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulNodeCreate,
		Update: resourceConsulNodeCreate,
		Read:   resourceConsulNodeRead,
		Delete: resourceConsulNodeDelete,

		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceConsulNodeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
	} else {
		var err error
		if dc, err = getDC(d, client); err != nil {
			return err
		}
	}

	var token string
	if v, ok := d.GetOk("token"); ok {
		token = v.(string)
	}

	// Setup the operations using the datacenter
	wOpts := consulapi.WriteOptions{Datacenter: dc, Token: token}

	address := d.Get("address").(string)
	name := d.Get("name").(string)

	registration := &consulapi.CatalogRegistration{
		Address:    address,
		Datacenter: dc,
		Node:       name,
	}

	if _, err := catalog.Register(registration, &wOpts); err != nil {
		return fmt.Errorf("Failed to register Consul catalog node with name '%s' at address '%s' in %s: %v",
			name, address, dc, err)
	}

	// Update the resource
	qOpts := consulapi.QueryOptions{Datacenter: dc}
	if _, _, err := catalog.Node(name, &qOpts); err != nil {
		return fmt.Errorf("Failed to read Consul catalog node with name '%s' at address '%s' in %s: %v",
			name, address, dc, err)
	} else {
		d.Set("datacenter", dc)
	}

	d.SetId(fmt.Sprintf("%s-%s", name, address))

	return nil
}

func resourceConsulNodeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	// Get the DC, error if not available.
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
	}

	name := d.Get("name").(string)

	// Setup the operations using the datacenter
	qOpts := consulapi.QueryOptions{Datacenter: dc}

	if _, _, err := catalog.Node(name, &qOpts); err != nil {
		return fmt.Errorf("Failed to get name '%s' from Consul catalog: %v", name, err)
	}

	return nil
}

func resourceConsulNodeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
	} else {
		var err error
		if dc, err = getDC(d, client); err != nil {
			return err
		}
	}

	var token string
	if v, ok := d.GetOk("token"); ok {
		token = v.(string)
	}

	// Setup the operations using the datacenter
	wOpts := consulapi.WriteOptions{Datacenter: dc, Token: token}

	address := d.Get("address").(string)
	name := d.Get("name").(string)

	deregistration := consulapi.CatalogDeregistration{
		Address:    address,
		Datacenter: dc,
		Node:       name,
	}

	if _, err := catalog.Deregister(&deregistration, &wOpts); err != nil {
		return fmt.Errorf("Failed to deregister Consul catalog node with name '%s' at address '%s' in %s: %v",
			name, address, dc, err)
	}

	// Clear the ID
	d.SetId("")
	return nil
}
