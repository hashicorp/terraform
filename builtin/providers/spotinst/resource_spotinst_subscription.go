package spotinst

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/spotinst/spotinst-sdk-go/spotinst"
)

func resourceSpotinstSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceSpotinstSubscriptionCreate,
		Update: resourceSpotinstSubscriptionUpdate,
		Read:   resourceSpotinstSubscriptionRead,
		Delete: resourceSpotinstSubscriptionDelete,

		Schema: map[string]*schema.Schema{
			"resource_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"event_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToUpper(value)
				},
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},

			"format": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
			},
		},
	}
}

func resourceSpotinstSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	newSubscription, err := buildSubscriptionOpts(d, meta)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Subscription create configuration: %#v\n", newSubscription)
	res, _, err := client.Subscription.Create(newSubscription)
	if err != nil {
		return fmt.Errorf("[ERROR] Error creating subscription: %s", err)
	}
	d.SetId(*res[0].ID)
	log.Printf("[INFO] Subscription created successfully: %s\n", d.Id())
	return resourceSpotinstSubscriptionRead(d, meta)
}

func resourceSpotinstSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	subscriptions, _, err := client.Subscription.Get(d.Id())
	if err != nil {
		if serr, ok := err.(*spotinst.ErrorResponse); ok {
			if serr.Response.StatusCode == 400 {
				d.SetId("")
				return nil
			} else {
				return fmt.Errorf("[ERROR] Error retrieving subscription: %s", err)
			}
		} else {
			return fmt.Errorf("[ERROR] Error retrieving subscription: %s", err)
		}
	}
	if len(subscriptions) == 0 {
		return fmt.Errorf("[ERROR] No matching subscription %s", d.Id())
	} else if len(subscriptions) > 1 {
		return fmt.Errorf("[ERROR] Got %d results, only one is allowed", len(subscriptions))
	} else if s := subscriptions[0]; s != nil {
		d.Set("resource_id", s.ResourceID)
		d.Set("event_type", s.EventType)
		d.Set("protocol", s.Protocol)
		d.Set("endpoint", s.Endpoint)
		d.Set("format", s.Format)
	} else {
		d.SetId("")
		return nil
	}
	return nil
}

func resourceSpotinstSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	hasChange := false
	client := meta.(*spotinst.Client)
	update := &spotinst.Subscription{ID: spotinst.String(d.Id())}

	if d.HasChange("resource_id") {
		update.ResourceID = spotinst.String(d.Get("resource_id").(string))
		hasChange = true
	}

	if d.HasChange("event_type") {
		update.EventType = spotinst.String(d.Get("event_type").(string))
		hasChange = true
	}

	if d.HasChange("protocol") {
		update.Protocol = spotinst.String(d.Get("protocol").(string))
		hasChange = true
	}

	if d.HasChange("endpoint") {
		update.Endpoint = spotinst.String(d.Get("endpoint").(string))
		hasChange = true
	}

	if d.HasChange("format") {
		update.Format = d.Get("format").(map[string]interface{})
		hasChange = true
	}

	if hasChange {
		log.Printf("[DEBUG] Subscription update configuration: %#v\n", update)
		_, _, err := client.Subscription.Update(update)
		if err != nil {
			return fmt.Errorf("[ERROR] Error updating subscription: %s", err)
		}
	}

	return resourceSpotinstSubscriptionRead(d, meta)
}

func resourceSpotinstSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting subscription: %s\n", d.Id())
	//client := meta.(*spotinst.Client)
	//subscription := &spotinst.Subscription{ID: d.Id()}
	//_, err := client.Subscription.Delete(subscription)
	//if err != nil {
	//	return fmt.Errorf("[ERROR] Error deleting subscription: %s", err)
	//}
	return nil
}

// buildSubscriptionOpts builds the Spotinst Subscription options.
func buildSubscriptionOpts(d *schema.ResourceData, meta interface{}) (*spotinst.Subscription, error) {
	subscription := &spotinst.Subscription{
		ResourceID: spotinst.String(d.Get("resource_id").(string)),
		EventType:  spotinst.String(strings.ToUpper(d.Get("event_type").(string))),
		Protocol:   spotinst.String(d.Get("protocol").(string)),
		Endpoint:   spotinst.String(d.Get("endpoint").(string)),
		Format:     d.Get("format").(map[string]interface{}),
	}

	return subscription, nil
}
