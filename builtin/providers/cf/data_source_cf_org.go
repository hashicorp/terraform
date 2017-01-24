package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceOrg() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceOrgRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceOrgRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	om := session.OrgManager()

	var (
		name string
		org  cfapi.CCOrg
	)

	name = d.Get("name").(string)

	org, err = om.FindOrg(name)

	if err != nil {
		return
	}
	d.SetId(org.ID)
	return
}
