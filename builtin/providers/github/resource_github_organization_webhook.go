package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubOrganizationWebhook() *schema.Resource {

	return &schema.Resource{
		Create: resourceGithubOrganizationWebhookCreate,
		Read:   resourceGithubOrganizationWebhookRead,
		Update: resourceGithubOrganizationWebhookUpdate,
		Delete: resourceGithubOrganizationWebhookDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateGithubOrganizationWebhookName,
			},
			"events": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"configuration": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func validateGithubOrganizationWebhookName(v interface{}, k string) (ws []string, errors []error) {
	if v.(string) != "web" {
		errors = append(errors, fmt.Errorf("Github: name can only be web"))
	}
	return
}

func resourceGithubOrganizationWebhookObject(d *schema.ResourceData) *github.Hook {
	url := d.Get("url").(string)
	active := d.Get("active").(bool)
	events := []string{}
	eventSet := d.Get("events").(*schema.Set)
	for _, v := range eventSet.List() {
		events = append(events, v.(string))
	}
	name := d.Get("name").(string)

	hook := &github.Hook{
		Name:   &name,
		URL:    &url,
		Events: events,
		Active: &active,
		Config: d.Get("configuration").(map[string]interface{}),
	}

	return hook
}

func resourceGithubOrganizationWebhookCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hk := resourceGithubOrganizationWebhookObject(d)

	hook, _, err := client.Organizations.CreateHook(context.TODO(), meta.(*Organization).name, hk)
	if err != nil {
		return err
	}
	d.SetId(strconv.Itoa(*hook.ID))

	return resourceGithubOrganizationWebhookRead(d, meta)
}

func resourceGithubOrganizationWebhookRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hookID, _ := strconv.Atoi(d.Id())

	hook, resp, err := client.Organizations.GetHook(context.TODO(), meta.(*Organization).name, hookID)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return err
	}
	d.Set("name", hook.Name)
	d.Set("url", hook.URL)
	d.Set("active", hook.Active)
	d.Set("events", hook.Events)
	d.Set("configuration", hook.Config)

	return nil
}

func resourceGithubOrganizationWebhookUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hk := resourceGithubOrganizationWebhookObject(d)
	hookID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	_, _, err = client.Organizations.EditHook(context.TODO(), meta.(*Organization).name, hookID, hk)
	if err != nil {
		return err
	}

	return resourceGithubOrganizationWebhookRead(d, meta)
}

func resourceGithubOrganizationWebhookDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hookID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Organizations.DeleteHook(context.TODO(), meta.(*Organization).name, hookID)
	return err
}
