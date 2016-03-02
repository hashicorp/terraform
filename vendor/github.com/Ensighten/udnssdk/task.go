package udnssdk

import (
	"fmt"
)

// ZonesService handles communication with the Zone related blah blah
type TasksService struct {
	client *Client
}

type Task struct {
	TaskId         string `json:"taskId"`
	TaskStatusCode string `json:"taskStatusCode"`
	Message        string `json:"message"`
	ResultUri      string `json:"resultUri"`
}

type TaskListDTO struct {
	Tasks                   []Task `json:"tasks"`
	Queryinfoq              string `json:"queryinfo/q"`
	Queryinfosort           string `json:"queryinfo/reverse"`
	Queryinfolimit          string `json:"queryinfo/limit"`
	ResultinfototalCount    string `json:"resultinfo/totalCount"`
	Resultinfooffset        string `json:"resultinfo/offset"`
	ResultinforeturnedCount string `json:"resultinfo/returnedCount"`
}
type taskWrapper struct {
	Task Task `json:"task"`
}

// taskPath links to the task url.
func taskResultPath(tid string) string {
	path := fmt.Sprintf("tasks/%s/result", tid)
	/*
		if tasktype != nil {
			path += fmt.Sprintf("/%v", tasktype)
			if task != nil {
				path += fmt.Sprintf("/%v", task)
			}
		}
	*/
	return path
}
func taskPath(tid string) string {
	return fmt.Sprintf("tasks/%s", tid)
}

// Get the status of a task.
func (s *TasksService) GetTaskStatus(tid string) (Task, *Response, error) {
	reqStr := taskPath(tid)
	var t Task
	res, err := s.client.get(reqStr, &t)
	if err != nil {
		return t, res, err
	}
	return t, res, err
}

// HTTP BS to dance around bad program structure
func (s *TasksService) GetTaskResultByURI(uri string) (*Response, error) {
	req, err := s.client.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client.HttpClient.Do(req)

	if err != nil {
		return &Response{Response: res}, err
	}
	return &Response{Response: res}, err
}

func (s *TasksService) GetTaskResult(tid string) (*Response, error) {
	uri := taskResultPath(tid)

	req, err := s.client.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	res, err := s.client.HttpClient.Do(req)

	if err != nil {
		return &Response{Response: res}, err
	}
	return &Response{Response: res}, err
}

// List tasks
//
func (s *TasksService) ListTasks(query string, offset, limit int) ([]Task, *Response, error) {
	// TODO: Soooo... This function does not handle pagination of Tasks....
	//v := url.Values{}

	reqStr := "tasks"
	var tld TaskListDTO
	//wrappedTasks := []Task{}

	res, err := s.client.get(reqStr, &tld)
	if err != nil {
		return []Task{}, res, err
	}

	tasks := []Task{}
	for _, t := range tld.Tasks {
		tasks = append(tasks, t)
	}

	return tasks, res, nil
}

// DeleteTask deletes a task.
//
func (s *TasksService) DeleteTask(tid string) (*Response, error) {
	path := taskPath(tid)
	return s.client.delete(path, nil)
}
