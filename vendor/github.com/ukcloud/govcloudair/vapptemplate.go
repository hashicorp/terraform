/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcloudair

import (
	"bytes"
	"encoding/xml"
	"fmt"
	types "github.com/ukcloud/govcloudair/types/v56"
)

type VAppTemplate struct {
	VAppTemplate *types.VAppTemplate
	c            *Client
}

func NewVAppTemplate(c *Client) *VAppTemplate {
	return &VAppTemplate{
		VAppTemplate: new(types.VAppTemplate),
		c:            c,
	}
}

func (v *Vdc) InstantiateVAppTemplate(template *types.InstantiateVAppTemplateParams) error {
	output, err := xml.MarshalIndent(template, "", "  ")
	if err != nil {
		return fmt.Errorf("Error finding VAppTemplate: %#v", err)
	}
	b := bytes.NewBufferString(xml.Header + string(output))

	s := v.c.VCDVDCHREF
	s.Path += "/action/instantiateVAppTemplate"

	req := v.c.NewRequest(map[string]string{}, "POST", s, b)
	req.Header.Add("Content-Type", "application/vnd.vmware.vcloud.instantiateVAppTemplateParams+xml")

	resp, err := checkResp(v.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error instantiating a new template: %s", err)
	}

	vapptemplate := NewVAppTemplate(v.c)
	if err = decodeBody(resp, vapptemplate.VAppTemplate); err != nil {
		return fmt.Errorf("error decoding orgvdcnetwork response: %s", err)
	}
	task := NewTask(v.c)
	for _, t := range vapptemplate.VAppTemplate.Tasks.Task {
		task.Task = t
		err = task.WaitTaskCompletion()
		if err != nil {
			return fmt.Errorf("Error performing task: %#v", err)
		}
	}
	return nil
}
