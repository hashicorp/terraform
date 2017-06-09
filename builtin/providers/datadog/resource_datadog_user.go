package datadog

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/zorkian/go-datadog-api.v2"
)

func resourceDatadogUser() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogUserCreate,
		Read:   resourceDatadogUserRead,
		Update: resourceDatadogUserUpdate,
		Delete: resourceDatadogUserDelete,
		Exists: resourceDatadogUserExists,
		Importer: &schema.ResourceImporter{
			State: resourceDatadogUserImport,
		},

		Schema: map[string]*schema.Schema{
			"disabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"email": {
				Type:     schema.TypeString,
				Required: true,
			},
			"handle": {
				Type:     schema.TypeString,
				Required: true,
			},
			"is_admin": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"verified": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceDatadogUserExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.
	client := meta.(*datadog.Client)

	if _, err := client.GetUser(d.Id()); err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceDatadogUserCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	var u datadog.User
	u.SetDisabled(d.Get("disabled").(bool))
	u.SetEmail(d.Get("email").(string))
	u.SetHandle(d.Get("handle").(string))
	u.SetIsAdmin(d.Get("is_admin").(bool))
	u.SetName(d.Get("name").(string))
	u.SetRole(d.Get("role").(string))

	// Datadog does not actually delete users, so CreateUser might return a 409.
	// We ignore that case and proceed, likely re-enabling the user.
	if _, err := client.CreateUser(u.Handle, u.Name); err != nil {
		if !strings.Contains(err.Error(), "API error 409 Conflict") {
			return fmt.Errorf("error creating user: %s", err.Error())
		}
		log.Printf("[INFO] Updating existing Datadog user %q", u.Handle)
	}

	if err := client.UpdateUser(u); err != nil {
		return fmt.Errorf("error creating user: %s", err.Error())
	}

	d.SetId(u.GetHandle())

	return resourceDatadogUserRead(d, meta)
}

func resourceDatadogUserRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	u, err := client.GetUser(d.Id())
	if err != nil {
		return err
	}

	d.Set("disabled", u.GetDisabled())
	d.Set("email", u.GetEmail())
	d.Set("handle", u.GetHandle())
	d.Set("is_admin", u.GetIsAdmin())
	d.Set("name", u.GetName())
	d.Set("role", u.GetRole())
	d.Set("verified", u.GetVerified())

	return nil
}

func resourceDatadogUserUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)
	var u datadog.User
	u.SetDisabled(d.Get("disabled").(bool))
	u.SetEmail(d.Get("email").(string))
	u.SetHandle(d.Id())
	u.SetIsAdmin(d.Get("is_admin").(bool))
	u.SetName(d.Get("name").(string))
	u.SetRole(d.Get("role").(string))

	if err := client.UpdateUser(u); err != nil {
		return fmt.Errorf("error updating user: %s", err.Error())
	}

	return resourceDatadogUserRead(d, meta)
}

func resourceDatadogUserDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	// Datadog does not actually delete users, but instead marks them as disabled.
	// Bypass DeleteUser if GetUser returns User.Disabled == true, otherwise it will 400.
	if u, err := client.GetUser(d.Id()); err == nil && u.GetDisabled() {
		return nil
	}

	if err := client.DeleteUser(d.Id()); err != nil {
		return err
	}

	return nil
}

func resourceDatadogUserImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := resourceDatadogUserRead(d, meta); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}
