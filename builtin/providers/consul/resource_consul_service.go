package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulService() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulServiceCreate,
		Update: resourceConsulServiceCreate,
		Read:   resourceConsulServiceRead,
		Delete: resourceConsulServiceDelete,

		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"service_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				ForceNew: true,
			},
		},
	}
}

func resourceConsulServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	agent := client.Agent()

	name := d.Get("name").(string)
	identifier := name

	if serviceId, ok := d.GetOk("service_id"); ok {
		identifier = serviceId.(string)
	}

	registration := consulapi.AgentServiceRegistration{Name: name, ID: identifier}

	if address, ok := d.GetOk("address"); ok {
		registration.Address = address.(string)
	}

	if port, ok := d.GetOk("port"); ok {
		registration.Port = port.(int)
	}

	if v, ok := d.GetOk("tags"); ok {
		vs := v.([]interface{})
		s := make([]string, len(vs))
		for i, raw := range vs {
			s[i] = raw.(string)
		}
		registration.Tags = s
	}

	if err := agent.ServiceRegister(&registration); err != nil {
		return fmt.Errorf("Failed to register service '%s' with Consul agent: %v", name, err)
	}

	// Update the resource
	if serviceMap, err := agent.Services(); err != nil {
		return fmt.Errorf("Failed to read services from Consul agent: %v", err)
	} else if service, ok := serviceMap[identifier]; !ok {
		return fmt.Errorf("Failed to read service '%s' from Consul agent: %v", identifier, err)
	} else {
		d.SetId(service.ID)

		d.Set("address", service.Address)
		d.Set("service_id", service.ID)
		d.Set("name", service.Service)
		d.Set("port", service.Port)
		tags := make([]string, 0, len(service.Tags))
		for _, tag := range service.Tags {
			tags = append(tags, tag)
		}
		d.Set("tags", tags)
	}

	return nil
}

func resourceConsulServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	agent := client.Agent()

	name := d.Get("name").(string)
	identifier := name

	if serviceId, ok := d.GetOk("service_id"); ok {
		identifier = serviceId.(string)
	}

	if services, err := agent.Services(); err != nil {
		return fmt.Errorf("Failed to get services from Consul agent: %v", err)
	} else if service, ok := services[identifier]; !ok {
		return fmt.Errorf("Failed to get service '%s' from Consul agent", identifier)
	} else {
		d.SetId(service.ID)

		d.Set("address", service.Address)
		d.Set("service_id", service.ID)
		d.Set("name", service.Service)
		d.Set("port", service.Port)
		tags := make([]string, 0, len(service.Tags))
		for _, tag := range service.Tags {
			tags = append(tags, tag)
		}
		d.Set("tags", tags)
	}

	return nil
}

func resourceConsulServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Agent()

	id := d.Get("service_id").(string)

	if err := catalog.ServiceDeregister(id); err != nil {
		return fmt.Errorf("Failed to deregister service '%s' from Consul agent: %v", id, err)
	}

	// Clear the ID
	d.SetId("")
	return nil
}
