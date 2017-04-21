package newrelic

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	newrelic "github.com/paultyng/go-newrelic/api"
)

func policyChannelExists(client *newrelic.Client, policyID int, channelID int) (bool, error) {
	channel, err := client.GetAlertChannel(channelID)
	if err != nil {
		if err == newrelic.ErrNotFound {
			return false, nil
		}

		return false, err
	}

	for _, id := range channel.Links.PolicyIDs {
		if id == policyID {
			return true, nil
		}
	}

	return false, nil
}

func resourceNewRelicAlertPolicyChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceNewRelicAlertPolicyChannelCreate,
		Read:   resourceNewRelicAlertPolicyChannelRead,
		// Update: Not currently supported in API
		Delete: resourceNewRelicAlertPolicyChannelDelete,
		Schema: map[string]*schema.Schema{
			"policy_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"channel_id": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNewRelicAlertPolicyChannelCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	policyID := d.Get("policy_id").(int)
	channelID := d.Get("channel_id").(int)

	serializedID := serializeIDs([]int{policyID, channelID})

	log.Printf("[INFO] Creating New Relic alert policy channel %s", serializedID)

	exists, err := policyChannelExists(client, policyID, channelID)
	if err != nil {
		return err
	}

	if !exists {
		err = client.UpdateAlertPolicyChannels(policyID, []int{channelID})
		if err != nil {
			return err
		}
	}

	d.SetId(serializedID)

	return nil
}

func resourceNewRelicAlertPolicyChannelRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	ids, err := parseIDs(d.Id(), 2)
	if err != nil {
		return err
	}

	policyID := ids[0]
	channelID := ids[1]

	log.Printf("[INFO] Reading New Relic alert policy channel %s", d.Id())

	exists, err := policyChannelExists(client, policyID, channelID)
	if err != nil {
		return err
	}

	if !exists {
		d.SetId("")
		return nil
	}

	d.Set("policy_id", policyID)
	d.Set("channel_id", channelID)

	return nil
}

func resourceNewRelicAlertPolicyChannelDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*newrelic.Client)

	ids, err := parseIDs(d.Id(), 2)
	if err != nil {
		return err
	}

	policyID := ids[0]
	channelID := ids[1]

	log.Printf("[INFO] Deleting New Relic alert policy channel %s", d.Id())

	exists, err := policyChannelExists(client, policyID, channelID)
	if err != nil {
		return err
	}

	if exists {
		if err := client.DeleteAlertPolicyChannel(policyID, channelID); err != nil {
			switch err {
			case newrelic.ErrNotFound:
				return nil
			}
			return err
		}
	}

	d.SetId("")

	return nil
}
