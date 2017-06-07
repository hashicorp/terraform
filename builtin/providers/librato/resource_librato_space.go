package librato

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/henrikhodne/go-librato/librato"
)

func resourceLibratoSpace() *schema.Resource {
	return &schema.Resource{
		Create: resourceLibratoSpaceCreate,
		Read:   resourceLibratoSpaceRead,
		Update: resourceLibratoSpaceUpdate,
		Delete: resourceLibratoSpaceDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceLibratoSpaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	name := d.Get("name").(string)

	space, _, err := client.Spaces.Create(&librato.Space{Name: librato.String(name)})
	if err != nil {
		return fmt.Errorf("Error creating Librato space %s: %s", name, err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Spaces.Get(*space.ID)
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	return resourceLibratoSpaceReadResult(d, space)
}

func resourceLibratoSpaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)

	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	space, _, err := client.Spaces.Get(uint(id))
	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error reading Librato Space %s: %s", d.Id(), err)
	}

	return resourceLibratoSpaceReadResult(d, space)
}

func resourceLibratoSpaceReadResult(d *schema.ResourceData, space *librato.Space) error {
	d.SetId(strconv.FormatUint(uint64(*space.ID), 10))
	if err := d.Set("id", *space.ID); err != nil {
		return err
	}
	if err := d.Set("name", *space.Name); err != nil {
		return err
	}
	return nil
}

func resourceLibratoSpaceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)
	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	if d.HasChange("name") {
		newName := d.Get("name").(string)
		log.Printf("[INFO] Modifying name space attribute for %d: %#v", id, newName)
		if _, err = client.Spaces.Update(uint(id), &librato.Space{Name: &newName}); err != nil {
			return err
		}
	}

	return resourceLibratoSpaceRead(d, meta)
}

func resourceLibratoSpaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*librato.Client)
	id, err := strconv.ParseUint(d.Id(), 10, 0)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Deleting Space: %d", id)
	_, err = client.Spaces.Delete(uint(id))
	if err != nil {
		if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
			log.Printf("Space %s not found", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error deleting space: %s", err)
	}

	resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, _, err := client.Spaces.Get(uint(id))
		if err != nil {
			if errResp, ok := err.(*librato.ErrorResponse); ok && errResp.Response.StatusCode == 404 {
				return nil
			}
			return resource.NonRetryableError(err)
		}
		return resource.RetryableError(fmt.Errorf("space still exists"))
	})

	d.SetId("")
	return nil
}
