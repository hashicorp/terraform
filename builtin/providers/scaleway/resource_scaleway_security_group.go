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
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceScalewaySecurityGroupCreate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	req := api.ScalewayNewSecurityGroup{
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
		Organization: scaleway.Organization,
	}

	err := scaleway.PostSecurityGroup(req)
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error creating security group: %q\n", serr.APIMessage)
		}

		return err
	}

	resp, err := scaleway.GetSecurityGroups()
	if err != nil {
		return err
	}

	for _, group := range resp.SecurityGroups {
		if group.Name == req.Name {
			d.SetId(group.ID)
			break
		}
	}

	if d.Id() == "" {
		return fmt.Errorf("Failed to find created security group.")
	}

	return resourceScalewaySecurityGroupRead(d, m)
}

func resourceScalewaySecurityGroupRead(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway
	resp, err := scaleway.GetASecurityGroup(d.Id())

	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] Error reading security group: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}

		return err
	}

	d.Set("name", resp.SecurityGroups.Name)
	d.Set("description", resp.SecurityGroups.Description)

	return nil
}

func resourceScalewaySecurityGroupUpdate(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	var req = api.ScalewayUpdateSecurityGroup{
		Organization: scaleway.Organization,
		Name:         d.Get("name").(string),
		Description:  d.Get("description").(string),
	}

	if err := scaleway.PutSecurityGroup(req, d.Id()); err != nil {
		log.Printf("[DEBUG] Error reading security group: %q\n", err)

		return err
	}

	return resourceScalewaySecurityGroupRead(d, m)
}

func resourceScalewaySecurityGroupDelete(d *schema.ResourceData, m interface{}) error {
	scaleway := m.(*Client).scaleway

	mu.Lock()
	defer mu.Unlock()

	err := scaleway.DeleteSecurityGroup(d.Id())
	if err != nil {
		if serr, ok := err.(api.ScalewayAPIError); ok {
			log.Printf("[DEBUG] error reading Security Group Rule: %q\n", serr.APIMessage)

			if serr.StatusCode == 404 {
				d.SetId("")
				return nil
			}
		}

		return err
	}

	d.SetId("")
	return nil
}
