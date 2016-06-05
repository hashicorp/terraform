package scaleway

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func resourceScalewaySecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceScalewaySecurityGroupCreate,
		Read:   resourceScalewaySecurityGroupRead,
		Update: resourceScalewaySecurityGroupUpdate,
		Delete: resourceScalewaySecurityGroupDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceScalewaySecurityGroupCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	def := api.ScalewayNewSecurityGroup{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		Organization: scaleway.Organization,
	}

	err := scaleway.PostSecurityGroup(def)
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("Error Posting Security Group. Reason: %s. %#v", serr.APIMessage, serr)
		}

		return serr
	}

	defs, e := scaleway.GetSecurityGroups()
	if e != nil {
		return e
	}
	for _, group := range defs.SecurityGroups {
		if group.Name == def.Name {
			d.SetId(group.ID)
			break
		}
	}
	if d.Id() == "" {
		return fmt.Errorf("Failed to find newly created security group.")
	}

	return resourceScalewaySecurityGroupRead(d, m)
}

func resourceScalewaySecurityGroupRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	groups, err := scaleway.GetASecurityGroup(d.Id())

	if err != nil {
		serr := err.(api.ScalewayAPIError)

		log.Printf("Error Reading Security Group. Reason: %s. %#v", serr.APIMessage, serr)

		if serr.StatusCode == 404 {
			d.SetId("")
			return nil
		}

		return serr
	}

	d.Set("name", groups.SecurityGroups.Name)
	d.Set("description", groups.SecurityGroups.Description)

	return nil
}

func resourceScalewaySecurityGroupUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	var def = api.ScalewayNewSecurityGroup{
		Organization: scaleway.Organization,
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
	}

	if err := scaleway.PutSecurityGroup(def, d.Id()); err != nil {
		serr := err.(api.ScalewayAPIError)

		log.Printf("Error Updating Security Group. Reason: %s. %#v", serr.APIMessage, serr)

		return err
	}

	return nil
}

func resourceScalewaySecurityGroupDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	err := scaleway.DeleteSecurityGroup(d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
