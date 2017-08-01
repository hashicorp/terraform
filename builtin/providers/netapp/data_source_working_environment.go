package netapp

import (
	"fmt"
	"log"

	"github.com/candidpartners/occm-sdk-go/api/workenv"
	"github.com/hashicorp/terraform/helper/schema"
)

type dataSourceWorkingEnvironmentsResult struct {
	*workenv.WorkingEnvironments
}

func dataSourceWorkingEnvironments() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceWorkingEnvironmentsRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourceWorkingEnvironmentsRead(d *schema.ResourceData, meta interface{}) error {
	apis := meta.(*APIs)

	log.Printf("[INFO] Reading working environment")

	workEnvName := d.Get("name").(string)

	resp, err := apis.WorkingEnvironmentAPI.GetWorkingEnvironments()
	if err != nil {
		return err
	}

	var found *workenv.VsaWorkingEnvironment

	for _, workEnv := range resp.VSA {
		if workEnv.Name == workEnvName {
			found = &workEnv
			break
		}
	}

	if found == nil {
		return fmt.Errorf("Working environment with name %s not found", workEnvName)
	}

	d.SetId(found.PublicId)
	d.Set("name", found.Name)

	return nil
}
