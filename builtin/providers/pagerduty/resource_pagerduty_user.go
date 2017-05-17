package pagerduty

import (
	"log"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePagerDutyUser() *schema.Resource {
	return &schema.Resource{
		Create: resourcePagerDutyUserCreate,
		Read:   resourcePagerDutyUserRead,
		Update: resourcePagerDutyUserUpdate,
		Delete: resourcePagerDutyUserDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"email": {
				Type:     schema.TypeString,
				Required: true,
			},
			"color": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"role": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "user",
				ValidateFunc: validateValueFunc([]string{
					"admin",
					"limited_user",
					"owner",
					"read_only_user",
					"team_responder",
					"user",
				}),
			},
			"job_title": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"avatar_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"teams": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"time_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"html_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"invitation_sent": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},
		},
	}
}

func buildUserStruct(d *schema.ResourceData) *pagerduty.User {
	user := pagerduty.User{
		Name:  d.Get("name").(string),
		Email: d.Get("email").(string),
		APIObject: pagerduty.APIObject{
			ID: d.Id(),
		},
	}

	if attr, ok := d.GetOk("color"); ok {
		user.Color = attr.(string)
	}

	if attr, ok := d.GetOk("role"); ok {
		role := attr.(string)
		// Skip setting the role if the user is the owner of the account.
		// Can't change this through the API.
		if role != "owner" {
			user.Role = role
		}
	}

	if attr, ok := d.GetOk("job_title"); ok {
		user.JobTitle = attr.(string)
	}

	if attr, ok := d.GetOk("description"); ok {
		user.Description = attr.(string)
	}

	return &user
}

func resourcePagerDutyUserCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	user := buildUserStruct(d)

	log.Printf("[INFO] Creating PagerDuty user %s", user.Name)

	user, err := client.CreateUser(*user)

	if err != nil {
		return err
	}

	d.SetId(user.ID)

	return resourcePagerDutyUserUpdate(d, meta)
}

func resourcePagerDutyUserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty user %s", d.Id())

	o := &pagerduty.GetUserOptions{}

	user, err := client.GetUser(d.Id(), *o)

	if err != nil {
		return err
	}

	d.Set("name", user.Name)
	d.Set("email", user.Email)
	d.Set("time_zone", user.Timezone)
	d.Set("color", user.Color)
	d.Set("role", user.Role)
	d.Set("avatar_url", user.AvatarURL)
	d.Set("description", user.Description)
	d.Set("job_title", user.JobTitle)
	d.Set("teams", user.Teams)

	return nil
}

func resourcePagerDutyUserUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	user := buildUserStruct(d)

	log.Printf("[INFO] Updating PagerDuty user %s", d.Id())

	if _, err := client.UpdateUser(*user); err != nil {
		return err
	}

	if d.HasChange("teams") {
		o, n := d.GetChange("teams")

		if o == nil {
			o = new(schema.Set)
		}

		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		remove := expandStringList(os.Difference(ns).List())
		add := expandStringList(ns.Difference(os).List())

		for _, t := range remove {
			_, tErr := client.GetTeam(t)

			if tErr != nil {
				log.Printf("[INFO] PagerDuty team: %s not found, removing dangling team reference for user %s", t, d.Id())
				continue
			}

			log.Printf("[INFO] Removing PagerDuty user %s from team: %s", d.Id(), t)

			rErr := client.RemoveUserFromTeam(t, d.Id())
			if rErr != nil {
				return rErr
			}
		}

		for _, t := range add {
			log.Printf("[INFO] Adding PagerDuty user %s to team: %s", d.Id(), t)

			aErr := client.AddUserToTeam(t, d.Id())
			if aErr != nil {
				return aErr
			}
		}
	}

	return nil
}

func resourcePagerDutyUserDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Deleting PagerDuty user %s", d.Id())

	if err := client.DeleteUser(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
