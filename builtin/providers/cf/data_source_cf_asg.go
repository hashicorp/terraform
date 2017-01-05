package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAsg() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceAsgRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceAsgRead(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	am := session.ASGManager()

	sg, err := am.Read(d.Get("name").(string))
	if err != nil {
		return err
	}
	d.SetId(sg.GUID)
	return nil
}
