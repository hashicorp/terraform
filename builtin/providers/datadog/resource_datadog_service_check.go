package datadog

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
)

// resourceDatadogServiceCheck is a Datadog monitor resource
func resourceDatadogServiceCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceDatadogServiceCheckCreate,
		Read:   resourceDatadogServiceCheckRead,
		Update: resourceDatadogServiceCheckUpdate,
		Delete: resourceDatadogServiceCheckDelete,
		Exists: resourceDatadogServiceCheckExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"check": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"check_count": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
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
	checkCount := d.Get("check_count").(string)

	// Examples queries
	// "http.can_connect".over("instance:buildeng_http","production").last(2).count_by_status()
	// "http.can_connect".over("*").by("host","instance","url").last(2).count_by_status()

	query = fmt.Sprintf("\"%s\".over(%s)%s.last(%s).count_by_status()", check, tagsParsed, keys, checkCount)
	log.Print(fmt.Sprintf("[DEBUG] submitting query: %s", query))
	monitorName = name

	o := datadog.Options{
		NotifyNoData:     d.Get("notify_no_data").(bool),
		NoDataTimeframe:  d.Get("no_data_timeframe").(int),
		RenotifyInterval: d.Get("renotify_interval").(int),
	}

	m := datadog.Monitor{
		Type:    "service check",
		Query:   query,
		Name:    monitorName,
		Message: fmt.Sprintf("%s", message),
		Options: o,
	}

	return &m
}

// resourceDatadogServiceCheckCreate creates a monitor.
func resourceDatadogServiceCheckCreate(d *schema.ResourceData, meta interface{}) error {
	log.Print("[DEBUG] creating monitor")
	client := meta.(*datadog.Client)

	log.Print("[DEBUG] Creating service check")
	m, err := client.CreateMonitor(buildServiceCheckStruct(d))

	if err != nil {
		return fmt.Errorf("error creating service check: %s", err)
	}

	d.SetId(strconv.Itoa(m.Id))
	return nil
}

// resourceDatadogServiceCheckDelete deletes a monitor.
func resourceDatadogServiceCheckDelete(d *schema.ResourceData, meta interface{}) error {
	log.Print("[DEBUG] deleting monitor")
	client := meta.(*datadog.Client)

	log.Print("[DEBUG] Deleting service check")
	ID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	err = client.DeleteMonitor(ID)
	if err != nil {
		return err
	}
	return nil
}

// resourceDatadogServiceCheckExists verifies a monitor exists.
func resourceDatadogServiceCheckExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.

	client := meta.(*datadog.Client)

	log.Print("[DEBUG] verifying service check exists")
	ID, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, err
	}

	_, err = client.GetMonitor(ID)

	if err != nil {
		if strings.EqualFold(err.Error(), "API error: 404 Not Found") {
			log.Printf("[DEBUG] Service Check does not exist: %s", err)
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// resourceDatadogServiceCheckRead synchronises Datadog and local state .
func resourceDatadogServiceCheckRead(d *schema.ResourceData, meta interface{}) error {
	// TODO: add support for this a read function.
	/* Read - This is called to resync the local state with the remote state.
	Terraform guarantees that an existing ID will be set. This ID should be
	used to look up the resource. Any remote data should be updated into the
	local data. No changes to the remote resource are to be made.
	*/

	return nil
}

// resourceDatadogServiceCheckUpdate updates a monitor.
func resourceDatadogServiceCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] running update.")

	client := meta.(*datadog.Client)

	body := buildServiceCheckStruct(d)

	ID, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	body.Id = ID
	err = client.UpdateMonitor(body)

	if err != nil {
		return fmt.Errorf("error updating warning: %s", err.Error())
	}
	return nil
}
