package pagerduty

import (
	"fmt"
	"log"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePagerDutyUser() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyUserRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"email": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataSourcePagerDutyUserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty user")

	searchEmail := d.Get("email").(string)

	o := &pagerduty.ListUsersOptions{
		Query: searchEmail,
	}

	resp, err := client.ListUsers(*o)
	if err != nil {
		return err
	}

	var found *pagerduty.User

	for _, user := range resp.Users {
		if user.Email == searchEmail {
			found = &user
			break
		}
	}

	if found == nil {
		return fmt.Errorf("Unable to locate any user with the email: %s", searchEmail)
	}

	d.SetId(found.ID)
	d.Set("name", found.Name)
	d.Set("email", found.Email)

	return nil
}
