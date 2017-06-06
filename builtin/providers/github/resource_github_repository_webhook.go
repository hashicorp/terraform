package github

import (
	"context"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceGithubRepositoryWebhook() *schema.Resource {
	return &schema.Resource{
		Create: resourceGithubRepositoryWebhookCreate,
		Read:   resourceGithubRepositoryWebhookRead,
		Update: resourceGithubRepositoryWebhookUpdate,
		Delete: resourceGithubRepositoryWebhookDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

func resourceGithubRepositoryWebhookObject(d *schema.ResourceData) *github.Hook {
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

func resourceGithubRepositoryWebhookCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hk := resourceGithubRepositoryWebhookObject(d)

	hook, _, err := client.Repositories.CreateHook(context.TODO(), meta.(*Organization).name, d.Get("repository").(string), hk)
	if err != nil {
		return err
	}
	d.SetId(strconv.Itoa(*hook.ID))

	return resourceGithubRepositoryWebhookRead(d, meta)
}

func resourceGithubRepositoryWebhookRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hookID, _ := strconv.Atoi(d.Id())

	hook, resp, err := client.Repositories.GetHook(context.TODO(), meta.(*Organization).name, d.Get("repository").(string), hookID)
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

func resourceGithubRepositoryWebhookUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hk := resourceGithubRepositoryWebhookObject(d)
	hookID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	_, _, err = client.Repositories.EditHook(context.TODO(), meta.(*Organization).name, d.Get("repository").(string), hookID, hk)
	if err != nil {
		return err
	}

	return resourceGithubRepositoryWebhookRead(d, meta)
}

func resourceGithubRepositoryWebhookDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Organization).client
	hookID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Repositories.DeleteHook(context.TODO(), meta.(*Organization).name, d.Get("repository").(string), hookID)
	return err
}
