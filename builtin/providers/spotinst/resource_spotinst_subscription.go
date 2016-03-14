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
		Read:   resourceSpotinstSubscriptionRead,
		Delete: resourceSpotinstSubscriptionDelete,

		Schema: map[string]*schema.Schema{
			"resource_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"event_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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
	d.SetId(res[0].ID)
	log.Printf("[INFO] Subscription created successfully: %s\n", d.Id())
	return resourceSpotinstSubscriptionRead(d, meta)
}

func resourceSpotinstSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*spotinst.Client)
	subscriptions, _, err := client.Subscription.Get(d.Id())
	if err != nil {
		serr, ok := err.(*spotinst.ErrorResponse)
		if ok {
			for _, r := range serr.Errors {
				if r.Code == "400" {
					d.SetId("")
					return nil
				} else {
					return fmt.Errorf("[ERROR] Error retrieving subscription: %s", err)
				}
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
		d.Set("event_type", strings.ToLower(s.EventType))
		d.Set("protocol", s.Protocol)
		d.Set("endpoint", s.Endpoint)
	} else {
		d.SetId("")
		return nil
	}
	return nil
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
		ResourceID: d.Get("resource_id").(string),
		EventType:  strings.ToUpper(d.Get("event_type").(string)),
		Protocol:   d.Get("protocol").(string),
		Endpoint:   d.Get("endpoint").(string),
	}

	return subscription, nil
}
