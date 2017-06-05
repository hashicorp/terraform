package heroku

import (
	"context"
	"log"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuAuthorization() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuAuthorizationCreate,
		Read:   resourceHerokuAuthorizationRead,
		Delete: resourceHerokuAuthorizationDelete,

		Schema: map[string]*schema.Schema{
			"scope": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
				ForceNew: true,
			},
			// --- Computed properties ---
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"token": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceHerokuAuthorizationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	authorization, err := client.OAuthAuthorizationInfo(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	d.SetId(authorization.ID)
	d.Set("id", authorization.ID)
	d.Set("scope", authorization.Scope)
	// TODO Missing in generated struct, but present in the API
	// d.Set("description", authorization.Description)
	if authorization.AccessToken != nil {
		d.Set("token", authorization.AccessToken.Token)
	} else {
		d.Set("token", nil)
	}

	return nil
}

func resourceHerokuAuthorizationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	description := d.Get("description").(string)
	scope := []string{}
	for _, v := range d.Get("scope").([]interface{}) {
		scope = append(scope, v.(string))
	}
	if len(scope) == 0 {
		scope = []string{"global"}
	}

	opts := heroku.OAuthAuthorizationCreateOpts{
		Description: &description,
		Scope:       scope,
	}
	if len(scope) > 0 {
		opts.Scope = scope
	}

	log.Printf("[DEBUG] OAuth Authorization configuration: %#v", opts)

	authorization, err := client.OAuthAuthorizationCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(authorization.ID)

	return resourceHerokuAuthorizationRead(d, meta)
}

func resourceHerokuAuthorizationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	id := d.Id()
	description := d.Get("description").(string)

	log.Printf("[INFO] Deleting authorization %s (%s)", id, description)

	_, err := client.OAuthAuthorizationDelete(context.TODO(), id)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
