package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
)

type Hook struct {
	Uuid        string   `json:"uuid,omitempty"`
	Url         string   `json:"url,omitempty"`
	Description string   `json:"description,omitempty"`
	Active      bool     `json:"active,omitempty"`
	Events      []string `json:"events,omitempty"`
}

func resourceHook() *schema.Resource {
	return &schema.Resource{
		Create: resourceHookCreate,
		Read:   resourceHookRead,
		Update: resourceHookUpdate,
		Delete: resourceHookDelete,
		Exists: resourceHookExists,

		Schema: map[string]*schema.Schema{
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"active": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"uuid": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"events": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func createHook(d *schema.ResourceData) *Hook {

	events := make([]string, 0, len(d.Get("events").(*schema.Set).List()))

	for _, item := range d.Get("events").(*schema.Set).List() {
		events = append(events, item.(string))
	}

	return &Hook{
		Url:         d.Get("url").(string),
		Description: d.Get("description").(string),
		Active:      d.Get("active").(bool),
		Events:      events,
	}
}

func resourceHookCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	hook := createHook(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(hook)

	hook_req, err := client.Post(fmt.Sprintf("2.0/repositories/%s/%s/hooks",
		d.Get("owner").(string),
		d.Get("repository").(string),
	), jsonpayload)

	decoder := json.NewDecoder(hook_req.Body)
	err = decoder.Decode(&hook)
	if err != nil {
		return err
	}

	d.SetId(hook.Uuid)
	d.Set("uuid", hook.Uuid)

	return resourceHookRead(d, m)
}
func resourceHookRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	hook_req, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s",
		d.Get("owner").(string),
		d.Get("repository").(string),
		d.Get("uuid").(string),
	))

	if err != nil {
		return err
	}

	var hook Hook

	decoder := json.NewDecoder(hook_req.Body)
	err = decoder.Decode(&hook)
	if err != nil {
		return err
	}

	d.Set("uuid", hook.Uuid)
	d.Set("description", hook.Description)
	d.Set("active", hook.Active)
	d.Set("url", hook.Url)

	eventsList := make([]string, 0, len(hook.Events))

	for _, event := range hook.Events {
		eventsList = append(eventsList, event)
	}

	d.Set("events", eventsList)

	return nil
}

func resourceHookUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	hook := createHook(d)

	var jsonbuffer []byte

	jsonpayload := bytes.NewBuffer(jsonbuffer)
	enc := json.NewEncoder(jsonpayload)
	enc.Encode(hook)

	hook_req, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s",
		d.Get("owner").(string),
		d.Get("repository").(string),
		d.Get("uuid").(string),
	), jsonpayload)

	if err != nil {
		return err
	}

	decoder := json.NewDecoder(hook_req.Body)
	err = decoder.Decode(&hook)
	if err != nil {
		return err
	}

	return resourceHookRead(d, m)
}

func resourceHookExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := m.(*BitbucketClient)
	if _, okay := d.GetOk("uuid"); okay {
		hook_req, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s",
			d.Get("owner").(string),
			d.Get("repository").(string),
			d.Get("uuid").(string),
		))

		if err != nil {
			panic(err)
		}

		if hook_req.StatusCode != 200 {
			d.SetId("")
			return false, nil
		}

		return true, nil
	} else {
		return false, nil
	}

}

func resourceHookDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*BitbucketClient)
	_, err := client.Delete(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s",
		d.Get("owner").(string),
		d.Get("repository").(string),
		d.Get("uuid").(string),
	))

	if err != nil {
		return err
	}
	return nil
}
