package librato

import (
	"fmt"
	"log"
	"reflect"
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
				StateFunc: normalizeJSON,
			},
		},
	}
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
		res, err := attributesExpand(normalizeJSON(v.(string)))
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

	log.Printf("[INFO] Reading Librato Service: %d", id)
	service, _, err := client.Services.Get(uint(id))
	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Librato Service %s: %s", d.Id(), err)
	}
	log.Printf("[INFO] Received Librato Service: %s", service)

	return resourceLibratoServiceReadResult(d, service)
}

func resourceLibratoServiceReadResult(d *schema.ResourceData, service *librato.Service) error {
	d.SetId(strconv.FormatUint(uint64(*service.ID), 10))
	d.Set("id", *service.ID)
	d.Set("type", *service.Type)
	d.Set("title", *service.Title)
	settings, _ := attributesFlatten(service.Settings)
	d.Set("settings", settings)

	return nil
}

func resourceLibratoServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	serviceID, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	// Just to have whole object for comparison before/after update
	fullService, _, err := client.Services.Get(uint(serviceID))
	if err != nil {
		return err
	}

	service := new(librato.Service)
	if d.HasChange("type") {
		service.Type = librato.String(d.Get("type").(string))
		fullService.Type = service.Type
	}
	if d.HasChange("title") {
		service.Title = librato.String(d.Get("title").(string))
		fullService.Title = service.Title
	}
	if d.HasChange("settings") {
		res, getErr := attributesExpand(normalizeJSON(d.Get("settings").(string)))
		if getErr != nil {
			return fmt.Errorf("Error expanding Librato service settings: %s", getErr)
		}
		service.Settings = res
		fullService.Settings = res
	}

	log.Printf("[INFO] Updating Librato Service %d: %s", serviceID, service)
	_, err = client.Services.Edit(uint(serviceID), service)
	if err != nil {
		return fmt.Errorf("Error updating Librato service: %s", err)
	}
	log.Printf("[INFO] Updated Librato Service %d", serviceID)

	// Wait for propagation since Librato updates are eventually consistent
	wait := resource.StateChangeConf{
		Pending:                   []string{fmt.Sprintf("%t", false)},
		Target:                    []string{fmt.Sprintf("%t", true)},
		Timeout:                   5 * time.Minute,
		MinTimeout:                2 * time.Second,
		ContinuousTargetOccurence: 5,
		Refresh: func() (interface{}, string, error) {
			log.Printf("[DEBUG] Checking if Librato Service %d was updated yet", serviceID)
			changedService, _, scErr := client.Services.Get(uint(serviceID))
			if scErr != nil {
				return changedService, "", scErr
			}
			isEqual := reflect.DeepEqual(*fullService, *changedService)
			log.Printf("[DEBUG] Updated Librato Service %d match: %t", serviceID, isEqual)
			return changedService, fmt.Sprintf("%t", isEqual), nil
		},
	}

	_, err = wait.WaitForState()
	if err != nil {
		return fmt.Errorf("Failed updating Librato Service %d: %s", serviceID, err)
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
