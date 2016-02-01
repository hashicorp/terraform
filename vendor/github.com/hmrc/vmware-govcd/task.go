/*
 * Copyright 2014 VMware, Inc.  All rights reserved.  Licensed under the Apache v2 License.
 */

package govcd

import (
	"fmt"
	"net/url"
	"time"

	types "github.com/hmrc/vmware-govcd/types/v56"
)

type Task struct {
	Task *types.Task
	c    *Client
}

func NewTask(c *Client) *Task {
	return &Task{
		Task: new(types.Task),
		c:    c,
	}
}

func (t *Task) Refresh() error {

	if t.Task == nil {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	u, _ := url.ParseRequestURI(t.Task.HREF)

	req := t.c.NewRequest(map[string]string{}, "GET", *u, nil)

	resp, err := checkResp(t.c.Http.Do(req))
	if err != nil {
		return fmt.Errorf("error retrieving task: %s", err)
	}

	// Empty struct before a new unmarshal, otherwise we end up with duplicate
	// elements in slices.
	t.Task = &types.Task{}

	if err = decodeBody(resp, t.Task); err != nil {
		return fmt.Errorf("error decoding task response: %s", err)
	}

	// The request was successful
	return nil
}

func (t *Task) WaitTaskCompletion() error {

	if t.Task == nil {
		return fmt.Errorf("cannot refresh, Object is empty")
	}

	for {
		err := t.Refresh()
		if err != nil {
			return fmt.Errorf("error retreiving task: %s", err)
		}

		// If task is not in a waiting status we're done, check if there's an error and return it.
		if t.Task.Status != "queued" && t.Task.Status != "preRunning" && t.Task.Status != "running" {
			if t.Task.Status == "error" {
				return fmt.Errorf("task did not complete succesfully: %s", t.Task.Description)
			}
			return nil
		}

		// Sleep for 3 seconds and try again.
		time.Sleep(3 * time.Second)
	}
}
