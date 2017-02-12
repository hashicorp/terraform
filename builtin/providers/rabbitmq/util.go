package rabbitmq

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func checkDeleted(d *schema.ResourceData, err error) error {
	if err.Error() == "not found" {
		d.SetId("")
		return nil
	}

	return err
}
