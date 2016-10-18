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
			State: resourcePagerDutyUserImport,
		},
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"email": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"color": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "user",
				ValidateFunc: validateValueFunc([]string{
					"admin",
					"limited_user",
					"owner",
					"read_only_user",
					"user",
				}),
			},
			"job_title": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"avatar_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"teams": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
			"time_zone": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"html_url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"invitation_sent": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},
			"description": &schema.Schema{
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
		user.Role = attr.(string)
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

	u := buildUserStruct(d)

	log.Printf("[INFO] Creating PagerDuty user %s", u.Name)

	u, err := client.CreateUser(*u)

	if err != nil {
		return err
	}

	d.SetId(u.ID)

	return resourcePagerDutyUserUpdate(d, meta)
}

func resourcePagerDutyUserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty user %s", d.Id())

	u, err := client.GetUser(d.Id(), pagerduty.GetUserOptions{})

	if err != nil {
		return err
	}

	d.Set("name", u.Name)
	d.Set("email", u.Email)
	d.Set("time_zone", u.Timezone)
	d.Set("color", u.Color)
	d.Set("role", u.Role)
	d.Set("avatar_url", u.AvatarURL)
	d.Set("description", u.Description)
	d.Set("job_title", u.JobTitle)
	d.Set("teams", u.Teams)

	return nil
}

func resourcePagerDutyUserUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	u := buildUserStruct(d)

	log.Printf("[INFO] Updating PagerDuty user %s", d.Id())

	u, err := client.UpdateUser(*u)

	if err != nil {
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

	err := client.DeleteUser(d.Id())

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyUserImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourcePagerDutyUserRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
