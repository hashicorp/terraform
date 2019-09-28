package test

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func testResourceTimeout() *schema.Resource {
	return &schema.Resource{
		Create: testResourceTimeoutCreate,
		Read:   testResourceTimeoutRead,
		Update: testResourceTimeoutUpdate,
		Delete: testResourceTimeoutDelete,

		// Due to the schema version also being stashed in the private/meta
		// data, we need to ensure that it does not overwrite the map
		// containing the timeouts.
		SchemaVersion: 1,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(time.Second),
			Update: schema.DefaultTimeout(time.Second),
			Delete: schema.DefaultTimeout(time.Second),
		},

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"create_delay": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"read_delay": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"update_delay": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"delete_delay": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func testResourceTimeoutCreate(d *schema.ResourceData, meta interface{}) error {
	delayString := d.Get("create_delay").(string)
	var delay time.Duration
	var err error
	if delayString != "" {
		delay, err = time.ParseDuration(delayString)
		if err != nil {
			return err
		}
	}

	if delay > d.Timeout(schema.TimeoutCreate) {
		return fmt.Errorf("timeout while creating resource")
	}

	d.SetId("testId")

	return testResourceRead(d, meta)
}

func testResourceTimeoutRead(d *schema.ResourceData, meta interface{}) error {
	delayString := d.Get("read_delay").(string)
	var delay time.Duration
	var err error
	if delayString != "" {
		delay, err = time.ParseDuration(delayString)
		if err != nil {
			return err
		}
	}

	if delay > d.Timeout(schema.TimeoutRead) {
		return fmt.Errorf("timeout while reading resource")
	}

	return nil
}

func testResourceTimeoutUpdate(d *schema.ResourceData, meta interface{}) error {
	delayString := d.Get("update_delay").(string)
	var delay time.Duration
	var err error
	if delayString != "" {
		delay, err = time.ParseDuration(delayString)
		if err != nil {
			return err
		}
	}

	if delay > d.Timeout(schema.TimeoutUpdate) {
		return fmt.Errorf("timeout while updating resource")
	}
	return nil
}

func testResourceTimeoutDelete(d *schema.ResourceData, meta interface{}) error {
	delayString := d.Get("delete_delay").(string)
	var delay time.Duration
	var err error
	if delayString != "" {
		delay, err = time.ParseDuration(delayString)
		if err != nil {
			return err
		}
	}

	if delay > d.Timeout(schema.TimeoutDelete) {
		return fmt.Errorf("timeout while deleting resource")
	}

	d.SetId("")
	return nil
}
