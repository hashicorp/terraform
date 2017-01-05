package opsgenie

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/opsgenie/opsgenie-go-sdk/user"
)

func dataSourceOpsGenieUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceOpsGenieUserRead,

		Schema: map[string]*schema.Schema{
			"username": {
				Type:     schema.TypeString,
				Required: true,
			},
			"full_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"role": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceOpsGenieUserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*OpsGenieClient).users

	username := d.Get("username").(string)

	log.Printf("[INFO] Reading OpsGenie user '%s'", username)

	o := user.ListUsersRequest{}
	resp, err := client.List(o)
	if err != nil {
		return nil
	}

	var found *user.GetUserResponse

	if len(resp.Users) > 0 {
		for _, user := range resp.Users {
			if user.Username == username {
				found = &user
				break
			}
		}
	}

	if found == nil {
		return fmt.Errorf("Unable to locate any user with the username: %s", username)
	}

	d.SetId(found.Id)
	d.Set("username", found.Username)
	d.Set("full_name", found.Fullname)
	d.Set("role", found.Role)

	return nil
}
