/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	types "github.com/hmrc/vmware-govcd/types/v56"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// OrgVDCNetwork an org vdc network client
type OrgVDCNetwork struct {
	OrgVDCNetwork *types.OrgVDCNetwork
	c             *Client
}

// NewOrgVDCNetwork creates an org vdc network client
func NewOrgVDCNetwork(c *Client) *OrgVDCNetwork {
	return &OrgVDCNetwork{
		OrgVDCNetwork: new(types.OrgVDCNetwork),
		c:             c,
	}
}

func (o *OrgVDCNetwork) Refresh() error {
	if o.OrgVDCNetwork.HREF == "" {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	u, _ := url.ParseRequestURI(o.OrgVDCNetwork.HREF)

	req := o.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(o.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retrieving task: %s", err)
	}

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	o.OrgVDCNetwork = &types.OrgVDCNetwork{}

	if err = decodeBody(resp, o.OrgVDCNetwork); err != nil {
		return fmt.Errorf("error decoding task response: %s", err)
	}

	// The request was successful
	return nil
}

func (o *OrgVDCNetwork) Delete() (Task, error) {
	err := o.Refresh()
	if err != nil {
		return Task{}, fmt.Errorf("Error refreshing network: %s", err)
	}
	pathArr := strings.Split(o.OrgVDCNetwork.HREF, "/")
	s, _ := url.ParseRequestURI(o.OrgVDCNetwork.HREF)
	s.Path = "/api/admin/network/" + pathArr[len(pathArr)-1]

	var resp *http.Response
	for {
		req := o.c.NewRequest(map[string]string{}, "DELETE", *s, nil)
		resp, err = checkResp(o.c.Http.Do(req))
		if err != nil {
			if v, _ := regexp.MatchString("is busy, cannot proceed with the operation.$", err.Error()); v {
				time.Sleep(3 * time.Second)
				continue
			}
			return Task{}, fmt.Errorf("error deleting Network: %s", err)
		}
		break
	}

	task := NewTask(o.c)

	if err = decodeBody(resp, task.Task); err != nil {
		return Task{}, fmt.Errorf("error decoding Task response: %s", err)
	}

	// The request was successful
	return *task, nil
}

func (v *Vdc) CreateOrgVDCNetwork(networkConfig *types.OrgVDCNetwork) error {
	for _, av := range v.Vdc.Link {
		if av.Rel == "add" && av.Type == "application/vnd.vmware.vcloud.orgVdcNetwork+xml" {
			u, err := url.ParseRequestURI(av.HREF)
			//return fmt.Errorf("Test output: %#v")

			if err != nil {
				return fmt.Errorf("error decoding vdc response: %s", err)
			}

			output, err := xml.MarshalIndent(networkConfig, "  ", "    ")
			if err != nil {
				return fmt.Errorf("error marshaling OrgVDCNetwork compose: %s", err)
			}

			//return fmt.Errorf("Test output: %s\n%#v", b, v.c)

			var resp *http.Response
			for {
				b := bytes.NewBufferString(xml.Header + string(output))
				log.Printf("[DEBUG] VCD Client configuration: %s", b)
				req := v.c.NewRequest(map[string]string{}, "POST", *u, b)
				req.Header.Add("Content-Type", av.Type)
				resp, err = checkResp(v.c.Http.Do(req))
				if err != nil {
					if v, _ := regexp.MatchString("is busy, cannot proceed with the operation.$", err.Error()); v {
						time.Sleep(3 * time.Second)
						continue
					}
					return fmt.Errorf("error instantiating a new OrgVDCNetwork: %s", err)
				}
				break
			}
			newstuff := NewOrgVDCNetwork(v.c)
			if err = decodeBody(resp, newstuff.OrgVDCNetwork); err != nil {
				return fmt.Errorf("error decoding orgvdcnetwork response: %s", err)
			}
			task := NewTask(v.c)
			for _, t := range newstuff.OrgVDCNetwork.Tasks.Task {
				task.Task = t
				err = task.WaitTaskCompletion()
				if err != nil {
					return fmt.Errorf("Error performing task: %#v", err)
				}
			}
		}
	}
	return nil
}
