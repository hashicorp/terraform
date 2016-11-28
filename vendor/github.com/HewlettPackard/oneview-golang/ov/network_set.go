package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type NetworkSet struct {
	Category              string          `json:"category,omitempty"`              // "category": "network-sets",
	ConnectionTemplateUri utils.Nstring   `json:"connectionTemplateUri,omitempty"` // "connectionTemplateUri": "/rest/connection-templates/7769cae0-b680-435b-9b87-9b864c81657f",
	Created               string          `json:"created,omitempty"`               // "created": "20150831T154835.250Z",
	Description           utils.Nstring   `json:"description,omitempty"`           // "description": "Network Set 1",
	ETAG                  string          `json:"eTag,omitempty"`                  // "eTag": "1441036118675/8",
	Modified              string          `json:"modified,omitempty"`              // "modified": "20150831T154835.250Z",
	Name                  string          `json:"name"`                            // "name": "Network Set 1",
	NativeNetworkUri      utils.Nstring   `json:"nativeNetworkUri,omitempty"`
	NetworkUris           []utils.Nstring `json:"networkUris"`
	State                 string          `json:"state,omitempty"`  // "state": "Normal",
	Status                string          `json:"status,omitempty"` // "status": "Critical",
	Type                  string          `json:"type"`             // "type": "network-set",
	URI                   utils.Nstring   `json:"uri,omitempty"`    // "uri": "/rest/network-set/e2f0031b-52bd-4223-9ac1-d91cb519d548"
}

type NetworkSetList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/network-sets?sort=name:asc"
	Members     []NetworkSet  `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetNetworkSetByName(name string) (NetworkSet, error) {
	var (
		netSet NetworkSet
	)
	netSets, err := c.GetNetworkSets(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if netSets.Total > 0 {
		return netSets.Members[0], err
	} else {
		return netSet, err
	}
}

func (c *OVClient) GetNetworkSets(filter string, sort string) (NetworkSetList, error) {
	var (
		uri         = "/rest/network-sets"
		q           map[string]interface{}
		networkSets NetworkSetList
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
		return networkSets, err
	}

	log.Debugf("GetNetworkSets %s", data)
	if err := json.Unmarshal([]byte(data), &networkSets); err != nil {
		return networkSets, err
	}
	return networkSets, nil
}

func (c *OVClient) CreateNetworkSet(netSet NetworkSet) error {
	log.Infof("Initializing creation of network set for %s.", netSet.Name)
	var (
		uri = "/rest/network-sets"
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, netSet)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, netSet)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new network set request: %s", err)
		return err
	}

	log.Debugf("Response New NetworkSet %s", data)
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

func (c *OVClient) DeleteNetworkSet(name string) error {
	var (
		netSet NetworkSet
		err    error
		t      *Task
		uri    string
	)

	netSet, err = c.GetNetworkSetByName(name)
	if err != nil {
		return err
	}
	if netSet.Name != "" {
		t = t.NewProfileTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", netSet.URI, netSet)
		log.Debugf("task -> %+v", t)
		uri = netSet.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete network set request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete network set %s", data)
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
		log.Infof("Network Set could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateNetworkSet(netSet NetworkSet) error {
	log.Infof("Initializing update of network set for %s.", netSet.Name)
	var (
		uri = netSet.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, netSet)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, netSet)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update network set request: %s", err)
		return err
	}

	log.Debugf("Response Update NetworkSet %s", data)
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
