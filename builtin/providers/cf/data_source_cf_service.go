package cloudfoundry

import (
	"fmt"

	"code.cloudfoundry.org/cli/cf/models"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceService() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceServiceRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"space": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func dataSourceServiceRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	sm := session.ServiceManager()

	var (
		name    string
		space   string
		service models.ServiceOffering
	)

	name = d.Get("name").(string)
	space = d.Get("space").(string)

	if len(space) == 0 {
		service, err = sm.FindServiceByName(name)
	} else {
		service, err = sm.FindSpaceService(name, space)
	}

	d.SetId(service.GUID)

	return
}
