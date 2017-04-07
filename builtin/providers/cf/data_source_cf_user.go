package cloudfoundry

import (
	"fmt"

	"code.cloudfoundry.org/cli/cf/models"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceUser() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceUserRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceUserRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	um := session.UserManager()

	var (
		name string
		user models.UserFields
	)

	name = d.Get("name").(string)

	user, err = um.FindByUsername(name)
	if err != nil {
		return
	}

	d.SetId(user.GUID)
	return
}
