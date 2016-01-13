package datadog

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

// resourceDatadogServiceCheck is a Datadog monitor resource
func resourceDatadogServiceCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogServiceCheckCreate,
		Read:   resourceDatadogGenericRead,
		Update: resourceDatadogServiceCheckUpdate,
		Delete: resourceDatadogGenericDelete,
		Exists: resourceDatadogGenericExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"check": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"thresholds": thresholdSchema(),

			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"keys": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"message": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Additional Settings
			"notify_no_data": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"no_data_timeframe": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"renotify_interval": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
		},
	}
}

// buildServiceCheckStruct returns a monitor struct
func buildServiceCheckStruct(d *schema.ResourceData) *datadog.Monitor {
	log.Print("[DEBUG] building monitor struct")
	name := d.Get("name").(string)
	message := d.Get("message").(string)

	// Tags are are no separate resource/gettable, so some trickery is needed
	var buffer bytes.Buffer
	if raw, ok := d.GetOk("tags"); ok {
		list := raw.([]interface{})
		length := (len(list) - 1)
		for i, v := range list {
			buffer.WriteString(fmt.Sprintf("\"%s\"", v))
			if i != length {
				buffer.WriteString(",")
			}

		}
	}

	tagsParsed := buffer.String()

	// Keys are used for multi alerts
	var b bytes.Buffer
	if raw, ok := d.GetOk("keys"); ok {
		list := raw.([]interface{})
		b.WriteString(".by(")
		length := (len(list) - 1)
		for i, v := range list {
			b.WriteString(fmt.Sprintf("\"%s\"", v))
			if i != length {
				b.WriteString(",")
			}

		}
		b.WriteString(")")
	}

	keys := b.String()

	var monitorName string
	var query string

	check := d.Get("check").(string)

	// Examples queries
	// "http.can_connect".over("instance:buildeng_http","production").last(2).count_by_status()
	// "http.can_connect".over("*").by("host","instance","url").last(2).count_by_status()

	checkCount, thresholds := getThresholds(d)

	query = fmt.Sprintf("\"%s\".over(%s)%s.last(%s).count_by_status()", check, tagsParsed, keys, checkCount)
	log.Print(fmt.Sprintf("[DEBUG] submitting query: %s", query))
	monitorName = name

	o := datadog.Options{
		NotifyNoData:     d.Get("notify_no_data").(bool),
		NoDataTimeframe:  d.Get("no_data_timeframe").(int),
		RenotifyInterval: d.Get("renotify_interval").(int),
		Thresholds:       thresholds,
	}

	m := datadog.Monitor{
		Type:    "service check",
		Query:   query,
		Name:    monitorName,
		Message: message,
		Options: o,
	}

	return &m
}

// resourceDatadogServiceCheckCreate creates a monitor.
func resourceDatadogServiceCheckCreate(d *schema.ResourceData, meta interface{}) error {
	log.Print("[DEBUG] creating monitor")

	m := buildServiceCheckStruct(d)
	if err := monitorCreator(d, meta, m); err != nil {
		return err
	}

	return nil
}

// resourceDatadogServiceCheckUpdate updates a monitor.
func resourceDatadogServiceCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] running update.")

	m := buildServiceCheckStruct(d)
	if err := monitorUpdater(d, meta, m); err != nil {
		return err
	}

	return nil
}
