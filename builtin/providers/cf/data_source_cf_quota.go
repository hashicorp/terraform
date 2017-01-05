package cloudfoundry

import (
	"fmt"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceQuota() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceQuotaRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"org": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func dataSourceQuotaRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	var (
		name, org string
		quota     cfapi.CCQuota
	)

	name = d.Get("name").(string)
	if v, ok := d.GetOk("org"); ok {
		org = v.(string)
	}

	qm := session.QuotaManager()
	if len(org) > 0 {
		quota, err = qm.FindSpaceQuota(name, org)
	} else {
		quota, err = qm.FindQuota(name)
	}
	if err != nil {
		return
	}
	d.SetId(quota.ID)
	return
}
