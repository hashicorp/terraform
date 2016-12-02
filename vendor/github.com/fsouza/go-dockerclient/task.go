// Copyright 2016 go-dockerclient authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package docker

import (
	"encoding/json"
	"net/http"

	"github.com/docker/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// NoSuchTask is the error returned when a given task does not exist.
type NoSuchTask struct {
	ID  string
	Err error
}

func (err *NoSuchTask) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return "No such task: " + err.ID
}

// ListTasksOptions specify parameters to the ListTasks function.
//
// See http://goo.gl/rByLzw for more details.
type ListTasksOptions struct {
	Filters map[string][]string
	Context context.Context
}

// ListTasks returns a slice of tasks matching the given criteria.
//
// See http://goo.gl/rByLzw for more details.
func (c *Client) ListTasks(opts ListTasksOptions) ([]swarm.Task, error) {
	path := "/tasks?" + queryString(opts)
	resp, err := c.do("GET", path, doOptions{context: opts.Context})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tasks []swarm.Task
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// InspectTask returns information about a task by its ID.
//
// See http://goo.gl/kyziuq for more details.
func (c *Client) InspectTask(id string) (*swarm.Task, error) {
	resp, err := c.do("GET", "/tasks/"+id, doOptions{})
	if err != nil {
		if e, ok := err.(*Error); ok && e.Status == http.StatusNotFound {
			return nil, &NoSuchTask{ID: id}
		}
		return nil, err
	}
	defer resp.Body.Close()
	var task swarm.Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, err
	}
	return &task, nil
}
