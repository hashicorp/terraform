package datadog

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/zorkian/go-datadog-api"
	"strconv"
	"strings"
)

func resourceDatadogGenericDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	if err = client.DeleteMonitor(i); err != nil {
		return err
	}

	return nil
}

func resourceDatadogGenericExists(d *schema.ResourceData, meta interface{}) (b bool, e error) {
	// Exists - This is called to verify a resource still exists. It is called prior to Read,
	// and lowers the burden of Read to be able to assume the resource exists.
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, err
	}

	if _, err = client.GetMonitor(i); err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceDatadogGenericRead(d *schema.ResourceData, meta interface{}) error {
	// TODO: Add support for read function.
	/* Read - This is called to resync the local state with the remote state.
	Terraform guarantees that an existing ID will be set. This ID should be
	used to look up the resource. Any remote data should be updated into the
	local data. No changes to the remote resource are to be made.
	*/

	return nil
}

func monitorCreator(d *schema.ResourceData, meta interface{}, m *datadog.Monitor) error {
	client := meta.(*datadog.Client)

	m, err := client.CreateMonitor(m)
	if err != nil {
		return fmt.Errorf("error updating montor: %s", err.Error())
	}

	d.SetId(strconv.Itoa(m.Id))

	return nil
}

func monitorUpdater(d *schema.ResourceData, meta interface{}, m *datadog.Monitor) error {
	client := meta.(*datadog.Client)

	i, err := strconv.Atoi(d.Id())
	if err != nil {
		return err
	}

	m.Id = i

	if err = client.UpdateMonitor(m); err != nil {
		return fmt.Errorf("error updating montor: %s", err.Error())
	}

	return nil
}
