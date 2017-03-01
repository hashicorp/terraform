package ns1

import (
	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/monitor"
)

func notifyListResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"notifications": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"config": &schema.Schema{
							Type:     schema.TypeMap,
							Required: true,
						},
					},
				},
			},
		},
		Create: NotifyListCreate,
		Read:   NotifyListRead,
		Update: NotifyListUpdate,
		Delete: NotifyListDelete,
	}
}

func notifyListToResourceData(d *schema.ResourceData, nl *monitor.NotifyList) error {
	d.SetId(nl.ID)
	d.Set("name", nl.Name)

	if len(nl.Notifications) > 0 {
		notifications := make([]map[string]interface{}, len(nl.Notifications))
		for i, n := range nl.Notifications {
			ni := make(map[string]interface{})
			ni["type"] = n.Type
			if n.Config != nil {
				ni["config"] = n.Config
			}
			notifications[i] = ni
		}
		d.Set("notifications", notifications)
	}
	return nil
}

func resourceDataToNotifyList(nl *monitor.NotifyList, d *schema.ResourceData) error {
	nl.ID = d.Id()

	if rawNotifications := d.Get("notifications").([]interface{}); len(rawNotifications) > 0 {
		ns := make([]*monitor.Notification, len(rawNotifications))
		for i, notificationRaw := range rawNotifications {
			ni := notificationRaw.(map[string]interface{})
			config := ni["config"].(map[string]interface{})

			switch ni["type"].(string) {
			case "webhook":
				ns[i] = monitor.NewWebNotification(config["url"].(string))
			case "email":
				ns[i] = monitor.NewEmailNotification(config["email"].(string))
			case "datafeed":
				ns[i] = monitor.NewFeedNotification(config["sourceid"].(string))
			}
		}
		nl.Notifications = ns
	}
	return nil
}

// NotifyListCreate creates an ns1 notifylist
func NotifyListCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)
	nl := monitor.NewNotifyList(d.Get("name").(string))

	if err := resourceDataToNotifyList(nl, d); err != nil {
		return err
	}

	if _, err := client.Notifications.Create(nl); err != nil {
		return err
	}

	return notifyListToResourceData(d, nl)
}

// NotifyListRead fetches info for the given notifylist from ns1
func NotifyListRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	nl, _, err := client.Notifications.Get(d.Id())
	if err != nil {
		return err
	}

	return notifyListToResourceData(d, nl)
}

// NotifyListDelete deletes the given notifylist from ns1
func NotifyListDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	_, err := client.Notifications.Delete(d.Id())
	d.SetId("")

	return err
}

// NotifyListUpdate updates the notifylist with given parameters
func NotifyListUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	nl := monitor.NewNotifyList(d.Get("name").(string))

	if err := resourceDataToNotifyList(nl, d); err != nil {
		return err
	}

	if _, err := client.Notifications.Update(nl); err != nil {
		return err
	}

	return notifyListToResourceData(d, nl)
}
