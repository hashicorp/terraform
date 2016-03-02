package ultradns

import (
	"fmt"
	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func schemaNotification() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{

			Schema: map[string]*schema.Schema{

				"ownerName": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"zoneName": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"recordType": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"email": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"poolRecords": &schema.Schema{
					Type:     schema.TypeList,
					Optional: false,
					Elem:     schemaPoolRecords(),
				},
			},
		},
	}
}
func schemaPoolRecords() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"poolRecord": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"notification": schemaNotificationInfo(),
			},
		},
	}
}
func schemaNotificationInfo() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"probe": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"record": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"scheduled": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}
func resourceUltraDNSNotificationCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	email := d.Get("email").(string)
	poolrecords := d.Get("poolRecords").(*schema.Set).List()

	name := d.Get("ownerName").(string)
	typ := d.Get("recordType").(string)
	zone := d.Get("zoneName").(string)

	var prs []udnssdk.NotificationPoolRecord

	for _, e := range poolrecords {
		vv := e.(*schema.ResourceData)
		prstr := vv.Get("poolrecord").(string)
		notif := vv.Get("notification").(*schema.ResourceData)
		nidto := udnssdk.NotificationInfoDTO{
			Probe:     notif.Get("probe").(bool),
			Record:    notif.Get("record").(bool),
			Scheduled: notif.Get("scheduled").(bool),
		}

		pr := udnssdk.NotificationPoolRecord{
			PoolRecord:   prstr,
			Notification: nidto,
		}
		prs = append(prs, pr)

	}

	newNotification := udnssdk.NotificationDTO{
		Email:       email,
		PoolRecords: prs,
	}
	log.Printf("[DEBUG] UltraDNS Notification create configuration: %#v", newNotification)

	r, err := client.SBTCService.CreateNotification(name, typ, zone, email, newNotification)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to create UltraDNS Notification: %s", err)
	}
	uri := r.Header.Get("Location")

	if err != nil {
		return fmt.Errorf("[ERROR] Failed to create UltraDNS Notification: %s", err)
	}
	d.Set("uri", uri)
	d.SetId(uri)

	log.Printf("[INFO] Notification ID: %s", d.Id())

	return resourceUltraDNSNotificationRead(d, meta)
}

func resourceUltraDNSNotificationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	notification, _, err := client.SBTCService.GetNotification(d.Get("name").(string), d.Get("type").(string), d.Get("zone").(string), d.Get("email").(string))

	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, r := range uderr.Responses {
				// 70002 means Notifications Not Found
				if r.ErrorCode == 70002 {
					d.SetId("")
					return nil
				} else {
					return fmt.Errorf("[ERROR] Couldn't find UltraDNS Notification: %s", err)
				}
			}
		} else {
			return fmt.Errorf("[ERROR] Couldn't find UltraDNS Notification: %s", err)
		}
	}
	//email := notification.Email
	var prs []map[string]interface{}
	poolrecords := notification.PoolRecords
	for _, e := range poolrecords {
		n := e.Notification
		pr := e.PoolRecord
		prs = append(prs, map[string]interface{}{"poolRecord": pr, "notification": n})
	}
	return nil
}

func resourceUltraDNSNotificationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	email := d.Get("email").(string)
	poolrecords := d.Get("poolRecords").(*schema.Set).List()

	name := d.Get("ownerName").(string)
	typ := d.Get("recordType").(string)
	zone := d.Get("zoneName").(string)

	var prs []udnssdk.NotificationPoolRecord

	for _, e := range poolrecords {
		vv := e.(*schema.ResourceData)
		prstr := vv.Get("poolrecord").(string)
		notif := vv.Get("notification").(*schema.ResourceData)
		nidto := udnssdk.NotificationInfoDTO{
			Probe:     notif.Get("probe").(bool),
			Record:    notif.Get("record").(bool),
			Scheduled: notif.Get("scheduled").(bool),
		}

		pr := udnssdk.NotificationPoolRecord{
			PoolRecord:   prstr,
			Notification: nidto,
		}
		prs = append(prs, pr)

	}

	updateNotification := udnssdk.NotificationDTO{
		Email:       email,
		PoolRecords: prs,
	}
	log.Printf("[DEBUG] UltraDNS Notification update configuration: %#v", updateNotification)

	_, err := client.SBTCService.UpdateNotification(name, typ, zone, email, updateNotification)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to create UltraDNS Notification: %s", err)
	}

	log.Printf("[INFO] Notification ID: %s", d.Id())

	log.Printf("[DEBUG] UltraDNS Notification update configuration: %#v", updateNotification)

	return resourceUltraDNSNotificationRead(d, meta)
}

func resourceUltraDNSNotificationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	zone := d.Get("zoneName").(string)
	email := d.Get("email").(string)
	name := d.Get("ownerName").(string)
	typ := d.Get("type").(string)
	log.Printf("[INFO] Deleting UltraDNS Notification: %s, %s", d.Get("zone").(string), email)

	_, err := client.SBTCService.DeleteNotification(name, typ, zone, email)

	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting UltraDNS Notification: %s", err)
	}

	return nil
}
