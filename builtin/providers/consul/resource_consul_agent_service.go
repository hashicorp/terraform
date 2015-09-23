package consul

import (
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceConsulAgentService() *schema.Resource {
	return &schema.Resource{
		Create: resourceConsulAgentServiceCreate,
		Update: resourceConsulAgentServiceCreate,
		Read:   resourceConsulAgentServiceRead,
		Delete: resourceConsulAgentServiceDelete,

		Schema: map[string]*schema.Schema{
			"address": &schema.Schema{
				Type:	 schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"name": &schema.Schema{
				Type: schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceConsulAgentServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	agent := client.Agent()

	name := d.Get("name").(string)

	registration := consulapi.AgentServiceRegistration{Name: name}

	if err := agent.ServiceRegister(&registration); err != nil {
		return fmt.Errorf("Failed to register service '%s' with Consul agent: %v", name, err)
	}

	// Update the resource
	d.SetId(fmt.Sprintf("consul-agent-service-%s", name))
	return nil
}

func resourceConsulAgentServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	agent := client.Agent()

	name := d.Get("name").(string)

	if services, err := agent.Services(); err != nil {
		return fmt.Errorf("Failed to get services from Consul agent: %v", err)
	} else {
		if _, ok := services[name]; !ok {
			return fmt.Errorf("Failed to get service '%s' from Consul agent: %v", name, err)
		}
	}

	return nil
}

func resourceConsulAgentServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*consulapi.Client)
	catalog := client.Agent()

	name := d.Get("name").(string)

	if err := catalog.ServiceDeregister(name); err != nil {
		return fmt.Errorf("Failed to deregister service '%s' from Consul agent: %v", name, err)
	}

	// Clear the ID
	d.SetId("")
	return nil
}
