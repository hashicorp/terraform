package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

//TODO change this struct to hold the variables from the GET API response body variables
type LogicalSwitchGroup struct {
	Category          string            `json:"category,omitempty"`    // "category": "logical-switch-groups",
	Created           string            `json:"created,omitempty"`     // "created": "20150831T154835.250Z",
	Description       utils.Nstring     `json:"description,omitempty"` // "description": "Logical Switch 1",
	ETAG              string            `json:"eTag,omitempty"`        // "eTag": "1441036118675/8",
	FabricUri         utils.Nstring     `json:"fabricUri,omitempty"`   // "fabricUri": "/rest/fabrics/9b8f7ec0-52b3-475e-84f4-c4eac51c2c20",
	Modified          string            `json:"modified,omitempty"`    // "modified": "20150831T154835.250Z",
	Name              string            `json:"name,omitempty"`        // "name": "Logical Switch Group1",
	State             string            `json:"state,omitempty"`       // "state": "Normal",
	Status            string            `json:"status,omitempty"`      // "status": "Critical",
	Type              string            `json:"type,omitempty"`        // "type": "logical-switch-groups",
	URI               utils.Nstring     `json:"uri,omitempty"`         // "uri": "/rest/logical-switch-groups/e2f0031b-52bd-4223-9ac1-d91cb519d548",
	SwitchMapTemplate SwitchMapTemplate `json:"switchMapTemplate"`
}

type LogicalSwitchGroupList struct {
	Total       int                  `json:"total,omitempty"`       // "total": 1,
	Count       int                  `json:"count,omitempty"`       // "count": 1,
	Start       int                  `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring        `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring        `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring        `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []LogicalSwitchGroup `json:"members,omitempty"`     // "members":[]
}

type SwitchMapTemplate struct {
	SwitchMapEntryTemplates []SwitchMapEntry `json:"switchMapEntryTemplates"`
}

type SwitchMapEntry struct {
	PermittedSwitchTypeUri utils.Nstring   `json:"permittedSwitchTypeUri"` //"permittedSwitchTypeUri": "/rest/switch-types/a2bc8f42-8bb8-4560-b80f-6c3c0e0d66e0",
	LogicalLocation        LogicalLocation `json:"logicalLocation"`
}

func (c *OVClient) GetLogicalSwitchGroupByName(name string) (LogicalSwitchGroup, error) {
	var (
		logicalSwitchGroup LogicalSwitchGroup
	)
	logicalSwitchGroups, err := c.GetLogicalSwitchGroups(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if logicalSwitchGroups.Total > 0 {
		return logicalSwitchGroups.Members[0], err
	} else {
		return logicalSwitchGroup, err
	}
}

func (c *OVClient) GetLogicalSwitchGroups(filter string, sort string) (LogicalSwitchGroupList, error) {
	var (
		uri                 = "/rest/logical-switch-groups"
		q                   map[string]interface{}
		logicalSwitchGroups LogicalSwitchGroupList
	)
	q = make(map[string]interface{})
	if len(filter) > 0 {
		q["filter"] = filter
	}

	if sort != "" {
		q["sort"] = sort
	}

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	// Setup query
	if len(q) > 0 {
		c.SetQueryString(q)
	}
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return logicalSwitchGroups, err
	}

	log.Debugf("GetLogicalSwitchGroups %s", data)
	if err := json.Unmarshal([]byte(data), &logicalSwitchGroups); err != nil {
		return logicalSwitchGroups, err
	}
	return logicalSwitchGroups, nil
}

func (c *OVClient) CreateLogicalSwitchGroup(logicalSwitchGroup LogicalSwitchGroup) error {
	log.Infof("Initializing creation of logicalSwitchGroup for %s.", logicalSwitchGroup.Name)
	var (
		uri = "/rest/logical-switch-groups"
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()

	log.Debugf("REST : %s \n %+v\n", uri, logicalSwitchGroup)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, logicalSwitchGroup)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new logical switch group request: %s", err)
		return err
	}

	log.Debugf("Response New LogicalSwitchGroup %s", data)
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		t.TaskIsDone = true
		log.Errorf("Error with task un-marshal: %s", err)
		return err
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (c *OVClient) DeleteLogicalSwitchGroup(name string) error {
	var (
		logicalSwitchGroup LogicalSwitchGroup
		err                error
		t                  *Task
		uri                string
	)

	logicalSwitchGroup, err = c.GetLogicalSwitchGroupByName(name)
	if err != nil {
		return err
	}
	if logicalSwitchGroup.Name != "" {
		t = t.NewProfileTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", logicalSwitchGroup.URI, logicalSwitchGroup)
		log.Debugf("task -> %+v", t)
		uri = logicalSwitchGroup.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete logicalSwitchGroup request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete logicalSwitchGroup %s", data)
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			t.TaskIsDone = true
			log.Errorf("Error with task un-marshal: %s", err)
			return err
		}
		err = t.Wait()
		if err != nil {
			return err
		}
		return nil
	} else {
		log.Infof("LogicalSwitchGroup could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateLogicalSwitchGroup(logicalSwitchGroup LogicalSwitchGroup) error {
	log.Infof("Initializing update of logical switch group for %s.", logicalSwitchGroup.Name)
	var (
		uri = logicalSwitchGroup.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, logicalSwitchGroup)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, logicalSwitchGroup)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update logical switch group request: %s", err)
		return err
	}

	log.Debugf("Response update LogicalSwitchGroup %s", data)
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		t.TaskIsDone = true
		log.Errorf("Error with task un-marshal: %s", err)
		return err
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return nil
}
