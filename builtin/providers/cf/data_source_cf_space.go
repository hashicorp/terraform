package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceSpace() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceSpaceRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"org": &schema.Schema{
				Type:     schema.TypeString,
				Required: true, // made required because it is not possible to look up a space with an org
			},
		},
	}
}

func dataSourceSpaceRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	sm := session.SpaceManager()

	var (
		name  string
		org   string
		space cfapi.CCSpace
	)

	name = d.Get("name").(string)
	org = d.Get("org").(string)
	space, err = sm.FindSpaceInOrg(name, org)
	if err != nil {
		return
	}

	d.SetId(space.ID)
	d.Set("name", space.Name)
	d.Set("org", space.OrgGUID)
	d.Set("quota", space.QuotaGUID)

	return
}
