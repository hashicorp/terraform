package consul

import (
	"bytes"
	"fmt"
	"log"

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
				Type:	 schema.TypeString,
				Required: true,
			},

			"datacenter": &schema.Schema{
				Type:	 schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"node": &schema.Schema{
				Type:	 schema.TypeString,
				Required: true,
			},

			"service": &schema.Schema{
				Type:	 schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource {
					Schema: map[string]*schema.Schema{
						"service": &schema.Schema{
							Type: schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceConsulCatalogEntryServicesHash,
			},

			"token": &schema.Schema{
				Type:	 schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceConsulCatalogEntryServicesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["service"].(string)))
	return hashcode.String(buf.String())
}

func resourceConsulCatalogEntryCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	// Resolve the datacenter first, all the other keys are dependent on this
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
		log.Printf("[DEBUG] Consul datacenter: %s", dc)
	} else {
		log.Printf("[DEBUG] Resolving Consul datacenter...")
		var err error
		dc, err = getDC(client)
		if err != nil {
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

	if rawServiceDefinition, ok := d.GetOk("service"); ok {
		rawServiceList := rawServiceDefinition.(*schema.Set).List()
		for _, rawService := range rawServiceList {
			service, ok := rawService.(map[string]interface{})

			if !ok {
				return fmt.Errorf("Failed to unroll: %#v", rawService)
			}

			serviceName := service["service"].(string)

			registration := consulapi.CatalogRegistration{
				Node: node, Address: address, Datacenter: dc,
				Service: &consulapi.AgentService{Service: serviceName},
			}

			if _, err := catalog.Register(&registration, &wOpts); err != nil {
				return fmt.Errorf("Failed to register Consul catalog entry with node '%s' at address '%s' with service %s in %s: %v",
					node, address, serviceName, dc, err)
			}
		}
	} else {
		registration := consulapi.CatalogRegistration{
			Node: node, Address: address, Datacenter: dc,
		}

		if _, err := catalog.Register(&registration, &wOpts); err != nil {
			return fmt.Errorf("Failed to register Consul catalog entry with node '%s' at address '%s' in %s: %v",
				node, address, dc, err)
		}
	}

	// Update the resource
	d.SetId(fmt.Sprintf("consul-catalog-node-%s-%s", node, address))
	d.Set("datacenter", dc)
	return nil
}

func resourceConsulCatalogEntryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Catalog()

	// Get the DC, error if not available.
	var dc string
	if v, ok := d.GetOk("datacenter"); ok {
		dc = v.(string)
		log.Printf("[DEBUG] Consul datacenter: %s", dc)
	} else {
		return fmt.Errorf("Missing datacenter configuration")
	}
	var token string
	if v, ok := d.GetOk("token"); ok {
		token = v.(string)
	}

	node := d.Get("node").(string)

	// Setup the operations using the datacenter
	qOpts := consulapi.QueryOptions{Datacenter: dc, Token: token}

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
		log.Printf("[DEBUG] Consul datacenter: %s", dc)
	} else {
		return fmt.Errorf("Missing datacenter configuration")
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

