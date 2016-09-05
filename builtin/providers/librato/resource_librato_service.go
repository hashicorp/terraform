package librato

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/henrikhodne/go-librato/librato"
)

func resourceLibratoService() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibratoServiceCreate,
		Read:   resourceLibratoServiceRead,
		Update: resourceLibratoServiceUpdate,
		Delete: resourceLibratoServiceDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"title": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"settings": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				StateFunc: normalizeJson,
			},
		},
	}
}

// Takes JSON in a string. Decodes JSON into
// settings hash
func resourceLibratoServicesExpandSettings(rawSettings string) (map[string]string, error) {
	var settings map[string]string

	settings = make(map[string]string)
	err := json.Unmarshal([]byte(rawSettings), &settings)
	if err != nil {
		return nil, fmt.Errorf("Error decoding JSON: %s", err)
	}

	return settings, err
}

// Encodes a settings hash into a JSON string
func resourceLibratoServicesFlatten(settings map[string]string) (string, error) {
	byteArray, err := json.Marshal(settings)
	if err != nil {
		return "", fmt.Errorf("Error encoding to JSON: %s", err)
	}

	return string(byteArray), nil
}

func normalizeJson(jsonString interface{}) string {
	if jsonString == nil || jsonString == "" {
		return ""
	}
	var j interface{}
	err := json.Unmarshal([]byte(jsonString.(string)), &j)
	if err != nil {
		return fmt.Sprintf("Error parsing JSON: %s", err)
	}
	b, _ := json.Marshal(j)
	return string(b[:])
}

func resourceLibratoServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	service := new(librato.Service)
	if v, ok := d.GetOk("type"); ok {
		service.Type = librato.String(v.(string))
	}
	if v, ok := d.GetOk("title"); ok {
		service.Title = librato.String(v.(string))
	}
	if v, ok := d.GetOk("settings"); ok {
		res, err := resourceLibratoServicesExpandSettings(normalizeJson(v.(string)))
		if err != nil {
			return fmt.Errorf("Error expanding Librato service settings: %s", err)
		}
		service.Settings = res
	}

	serviceResult, _, err := client.Services.Create(service)

	if err != nil {
		return fmt.Errorf("Error creating Librato service: %s", err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Services.Get(*serviceResult.ID)
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	return resourceLibratoServiceReadResult(d, serviceResult)
}

func resourceLibratoServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)
	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	service, _, err := client.Services.Get(uint(id))
	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Librato Service %s: %s", d.Id(), err)
	}

	return resourceLibratoServiceReadResult(d, service)
}

func resourceLibratoServiceReadResult(d *schema.ResourceData, service *librato.Service) error {
	d.SetId(strconv.FormatUint(uint64(*service.ID), 10))
	d.Set("id", *service.ID)
	d.Set("type", *service.Type)
	d.Set("title", *service.Title)
	settings, _ := resourceLibratoServicesFlatten(service.Settings)
	d.Set("settings", settings)

	return nil
}

func resourceLibratoServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	serviceID, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	service := new(librato.Service)
	if d.HasChange("type") {
		service.Type = librato.String(d.Get("type").(string))
	}
	if d.HasChange("title") {
		service.Title = librato.String(d.Get("title").(string))
	}
	if d.HasChange("settings") {
		res, err := resourceLibratoServicesExpandSettings(normalizeJson(d.Get("settings").(string)))
		if err != nil {
			return fmt.Errorf("Error expanding Librato service settings: %s", err)
		}
		service.Settings = res
	}

	_, err = client.Services.Edit(uint(serviceID), service)
	if err != nil {
		return fmt.Errorf("Error updating Librato service: %s", err)
	}

	return resourceLibratoServiceRead(d, meta)
}

func resourceLibratoServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)
	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting Service: %d", id)
	_, err = client.Services.Delete(uint(id))
	if err != nil {
		return fmt.Errorf("Error deleting Service: %s", err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Services.Get(uint(id))
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return resource.RetryableError(fmt.Errorf("service still exists"))
	})

	d.SetId("")
	return nil
}
