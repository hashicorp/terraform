package external

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSource() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceRead,

		Schema: map[string]*schema.Schema{
			"program": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"query": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"result": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceRead(d *schema.ResourceData, meta interface{}) error {

	programI := d.Get("program").([]interface{})
	query := d.Get("query").(map[string]interface{})

	// This would be a ValidateFunc if helper/schema allowed these
	// to be applied to lists.
	if err := validateProgramAttr(programI); err != nil {
		return err
	}

	program := make([]string, len(programI))
	for i, vI := range programI {
		program[i] = vI.(string)
	}

	cmd := exec.Command(program[0], program[1:]...)

	queryJson, err := json.Marshal(query)
	if err != nil {
		// Should never happen, since we know query will always be a map
		// from string to string, as guaranteed by d.Get and our schema.
		return err
	}

	cmd.Stdin = bytes.NewReader(queryJson)

	resultJson, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.Stderr != nil && len(exitErr.Stderr) > 0 {
				return fmt.Errorf("failed to execute %q: %s", program[0], string(exitErr.Stderr))
			}
			return fmt.Errorf("command %q failed with no error message", program[0])
		} else {
			return fmt.Errorf("failed to execute %q: %s", program[0], err)
		}
	}

	result := map[string]string{}
	err = json.Unmarshal(resultJson, &result)
	if err != nil {
		return fmt.Errorf("command %q produced invalid JSON: %s", program[0], err)
	}

	d.Set("result", result)

	d.SetId("-")
	return nil
}
