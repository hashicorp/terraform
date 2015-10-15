package consul

import (
	"bytes"
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulCatalogEntry() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulCatalogEntryCreate,
		Update: resourceConsulCatalogEntryCreate,
		Read:   resourceConsulCatalogEntryRead,
		Delete: resourceConsulCatalogEntryDelete,

		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"node": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"service": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},

						"tags": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				Set: resourceConsulCatalogEntryServicesHash,
			},

			"token": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceConsulCatalogEntryServicesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["id"].(string)))
	return hashcode.String(buf.String())
}

func resourceConsulCatalogEntryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
	} else {
		var err error
		if dc, err = getDC(client); err != nil {
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
	node := d.Get("node").(string)

	if services, ok := d.GetOk("service"); ok {
		for _, rawService := range services.(*schema.Set).List() {
			serviceData := rawService.(map[string]interface{})

			rawTags := serviceData["tags"].([]interface{})
			tags := make([]string, len(rawTags))
			for i, v := range rawTags {
				tags[i] = v.(string)
			}

			registration := consulapi.CatalogRegistration{
				Address:    address,
				Datacenter: dc,
				Node:       node,
				Service: &consulapi.AgentService{
					Address: serviceData["address"].(string),
					ID:      serviceData["id"].(string),
					Service: serviceData["name"].(string),
					Port:    serviceData["port"].(int),
					Tags:    tags,
				},
			}

			if _, err := catalog.Register(&registration, &wOpts); err != nil {
				return fmt.Errorf("Failed to register Consul catalog entry with node '%s' at address '%s' in %s: %v",
					node, address, dc, err)
			}
		}
	} else {
		registration := consulapi.CatalogRegistration{
			Address:    address,
			Datacenter: dc,
			Node:       node,
		}

		if _, err := catalog.Register(&registration, &wOpts); err != nil {
			return fmt.Errorf("Failed to register Consul catalog entry with node '%s' at address '%s' in %s: %v",
				node, address, dc, err)
		}
	}

	// Update the resource
	qOpts := consulapi.QueryOptions{Datacenter: dc}
	if _, _, err := catalog.Node(node, &qOpts); err != nil {
		return fmt.Errorf("Failed to read Consul catalog entry for node '%s' at address '%s' in %s: %v",
			node, address, dc, err)
	} else {
		d.Set("datacenter", dc)
	}
	d.SetId(fmt.Sprintf("consul-catalog-node-%s-%s", node, address))
	return nil
}

func resourceConsulCatalogEntryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	// Get the DC, error if not available.
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
	}

	node := d.Get("node").(string)

	// Setup the operations using the datacenter
	qOpts := consulapi.QueryOptions{Datacenter: dc}

	if _, _, err := catalog.Node(node, &qOpts); err != nil {
		return fmt.Errorf("Failed to get node '%s' from Consul catalog: %v", node, err)
	}

	return nil
}

func resourceConsulCatalogEntryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	// Get the DC, error if not available.
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
	}

	var token string
	if v, ok := d.GetOk("token"); ok {
		token = v.(string)
	}

	// Setup the operations using the datacenter
	wOpts := consulapi.WriteOptions{Datacenter: dc, Token: token}

	address := d.Get("address").(string)
	node := d.Get("node").(string)

	deregistration := consulapi.CatalogDeregistration{
		Node: node, Address: address, Datacenter: dc,
	}

	if _, err := catalog.Deregister(&deregistration, &wOpts); err != nil {
		return fmt.Errorf("Failed to deregister Consul catalog entry with node '%s' at address '%s' in %s: %v",
			node, address, dc, err)
	}

	// Clear the ID
	d.SetId("")
	return nil
}
