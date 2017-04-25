package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackAffinityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackAffinityGroupCreate,
		Read:   resourceCloudStackAffinityGroupRead,
		Delete: resourceCloudStackAffinityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackAffinityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	name := d.Get("name").(string)
	affinityGroupType := d.Get("type").(string)

	// Create a new parameter struct
	p := cs.AffinityGroup.NewCreateAffinityGroupParams(name, affinityGroupType)

	// Set the description
	if description, ok := d.GetOk("description"); ok {
		p.SetDescription(description.(string))
	} else {
		p.SetDescription(name)
	}

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating affinity group %s", name)
	r, err := cs.AffinityGroup.CreateAffinityGroup(p)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Affinity group %s successfully created", name)
	d.SetId(r.Id)

	return resourceCloudStackAffinityGroupRead(d, meta)
}

func resourceCloudStackAffinityGroupRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	log.Printf("[DEBUG] Rerieving affinity group %s", d.Get("name").(string))

	// Get the affinity group details
	ag, count, err := cs.AffinityGroup.GetAffinityGroupByID(
		d.Id(),
		cloudstack.WithProject(d.Get("project").(string)),
	)
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Affinity group %s does not longer exist", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	// Update the config
	d.Set("name", ag.Name)
	d.Set("description", ag.Description)
	d.Set("type", ag.Type)

	return nil
}

func resourceCloudStackAffinityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.AffinityGroup.NewDeleteAffinityGroupParams()
	p.SetId(d.Id())

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	// Delete the affinity group
	_, err := cs.AffinityGroup.DeleteAffinityGroup(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting affinity group: %s", err)
	}

	return nil
}
