package newrelic

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	newrelic "github.com/paultyng/go-newrelic/api"
)

var alertChannelTypes = map[string][]string{
	"campfire": []string{
		"room",
		"subdomain",
		"token",
	},
	"email": []string{
		"include_json_attachment",
		"recipients",
	},
	"hipchat": []string{
		"auth_token",
		"base_url",
		"room_id",
	},
	"opsgenie": []string{
		"api_key",
		"recipients",
		"tags",
		"teams",
	},
	"pagerduty": []string{
		"service_key",
	},
	"slack": []string{
		"channel",
		"url",
	},
	"user": []string{
		"user_id",
	},
	"victorops": []string{
		"key",
		"route_key",
	},
	"webhook": []string{
		"auth_password",
		"auth_type",
		"auth_username",
		"base_url",
		"headers",
		"payload_type",
		"payload",
	},
}

func resourceNewRelicAlertChannel() *schema.Resource {
	validAlertChannelTypes := make([]string, 0, len(alertChannelTypes))
	for k := range alertChannelTypes {
		validAlertChannelTypes = append(validAlertChannelTypes, k)
	}

	return &schema.Resource{
		Create: resourceNewRelicAlertChannelCreate,
		Read:   resourceNewRelicAlertChannelRead,
		// Update: Not currently supported in API
		Delete: resourceNewRelicAlertChannelDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(validAlertChannelTypes, false),
			},
			"configuration": {
				Type:     schema.TypeMap,
				Required: true,
				ForceNew: true,
				//TODO: ValidateFunc: (use list of keys from map above)
				Sensitive: true,
			},
		},
	}
}

func buildAlertChannelStruct(d *schema.ResourceData) *newrelic.AlertChannel {
	channel := newrelic.AlertChannel{
		Name:          d.Get("name").(string),
		Type:          d.Get("type").(string),
		Configuration: d.Get("configuration").(map[string]interface{}),
	}

	return &channel
}

func resourceNewRelicAlertChannelCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)
	channel := buildAlertChannelStruct(d)

	log.Printf("[INFO] Creating New Relic alert channel %s", channel.Name)

	channel, err := client.CreateAlertChannel(*channel)
	if err != nil {
		return err
	}

	d.SetId(strconv.Itoa(channel.ID))

	return nil
}

func resourceNewRelicAlertChannelRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Reading New Relic alert channel %v", id)

	channel, err := client.GetAlertChannel(int(id))
	if err != nil {
		if err == newrelic.ErrNotFound {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", channel.Name)
	d.Set("type", channel.Type)
	if err := d.Set("configuration", channel.Configuration); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Alert Channel Configuration: %#v", err)
	}

	return nil
}

func resourceNewRelicAlertChannelDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	id, err := strconv.ParseInt(d.Id(), 10, 32)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting New Relic alert channel %v", id)

	if err := client.DeleteAlertChannel(int(id)); err != nil {
		return err
	}

	d.SetId("")

	return nil
}
