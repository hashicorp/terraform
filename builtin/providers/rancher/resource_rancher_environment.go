package rancher

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceRancherEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherEnvironmentCreate,
		Read:   resourceRancherEnvironmentRead,
		Delete: resourceRancherEnvironmentDelete,
		Exists: resourceRancherEnvironmentExists,

		// http://docs.rancher.com/rancher/v1.2/en/api/api-resources/project/
		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"kubernetes": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"members": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"mesos": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"public_dns": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"services_port_range": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					"start_port": &schema.Schema{
						Type:     schema.TypeInt,
						Required: true,
					},

					"end_port": &schema.Schema{
						Type:     schema.TypeInt,
						Required: true,
					},
				},
			},

			"swarm": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"virtual_machine": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceRancherEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	env := Environment{
		Name: d.Get("name").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		env.Description = v.(string)
	}

	if v, ok := d.GetOk("kubernetes"); ok {
		env.Kubernetes = v.(bool)
	}

	if v, ok := d.GetOk("members"); ok {
		//	Members:     []EnvironmentMember{},
	}

	if v, ok := d.GetOk("mesos"); ok {
		env.Mesos = v.(bool)
	}

	if v, ok := d.GetOk("public_dns"); ok {
		env.PublicDNS = v.(bool)
	}

	if v, ok := d.GetOk("services_port_range"); ok {
		portRange := v.(map[string]int)
		env.ServicesPortRange = PortRange{
			StartPort: portRange["start_port"],
			EndPort:   portRange["end_port"],
		}
	}

	if v, ok := d.GetOk("swarm"); ok {
		env.Swarm = v.(bool)
	}

	if v, ok := d.GetOk("virtual_machine"); ok {
		env.VirtualMachine = v.(bool)
	}

	log.Printf("[DEBUG] Creating Rancher Environment: %#v", env)
	id, err := client.CreateEnvironment(env)
	if err != nil {
		return fmt.Errorf("Failed to create Rancher Environment: %s", err)
	}

	d.SetId(id)
	return resourceRancherEnvironmentRead(d, meta)
}

func resourceRancherEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	log.Printf("[DEBUG] Reading Rancher Environment: %s", d.Id())
	env, err := client.GetEnvironmentById(d.Id())
	if err != nil {
		return fmt.Errorf("Couldn't fetch Rancher Environment: %s", err)
	}

	// Set stuff here?

	return nil
}

func resourceRancherEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	log.Printf("[INFO] Deleting Rancher Environment: %s", d.Id())
	err := client.DeleteEnvironementByID(d.Id())

	if err != nil {
		return fmt.Errorf("Error deleting Rancher Environment: %s", err)
	}

	return nil
}

func resourceRancherEnvironmentExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	name := d.Get("name").(string)

	log.Printf("[INFO] Checking existence of Rancher Environment: %s", name)

	client := meta.(*Client)
	exists, err := client.EnvironmentExists(name)

	if err != nil {
		return false, fmt.Errorf("Error checking Rancher Environment: %s", err)
	} else {
		return exists, nil
	}
}
