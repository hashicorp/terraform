package netapp

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceCloudWorkingEnvironment() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceCloudWorkingEnvironmentRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"public_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"svm_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"is_ha": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceCloudWorkingEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	apis := meta.(*APIs)

	workEnvName := d.Get("name").(string)

	workenv, err := GetWorkingEnvironmentByName(apis, workEnvName)
	if err != nil {
		return err
	}

	d.SetId(workenv.PublicId)
	d.Set("public_id", workenv.PublicId)
	d.Set("name", workenv.Name)
	d.Set("tenant_id", workenv.TenantId)
	d.Set("svm_name", workenv.SvmName)
	d.Set("is_ha", workenv.IsHA)

	return nil
}
