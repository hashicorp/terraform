package ov

import (
	"encoding/json"
	"fmt"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type FCNetwork struct {
	Type                    string        `json:"type,omitempty"`
	FabricType              string        `json:"fabricType,omitempty"`
	FabricUri               utils.Nstring `json:"fabricUri,omitempty"`
	ConnectionTemplateUri   utils.Nstring `json:"connectionTemplateUri,omitempty"`
	ManagedSanURI           utils.Nstring `json:"managedSanUri,omitempty"`
	LinkStabilityTime       int           `json:"linkStabilityTime,omitempty"`
	AutoLoginRedistribution bool          `json:"autoLoginRedistribution"`
	Description             string        `json:"description,omitempty"`
	Name                    string        `json:"name,omitempty"`
	State                   string        `json:"state,omitempty"`
	Status                  string        `json:"status,omitempty"`
	Category                string        `json:"category,omitempty"`
	URI                     utils.Nstring `json:"uri,omitempty"`
	ETAG                    string        `json:"eTag,omitempty"`
	Modified                string        `json:"modified,omitempty"`
	Created                 string        `json:"created,omitempty"`
}

type FCNetworkList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []FCNetwork   `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetFCNetworkByName(name string) (FCNetwork, error) {
	var (
		fcNet FCNetwork
	)
	fcNets, err := c.GetFCNetworks(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if fcNets.Total > 0 {
		return fcNets.Members[0], err
	} else {
		return fcNet, err
	}
}

func (c *OVClient) GetFCNetworks(filter string, sort string) (FCNetworkList, error) {
	var (
		uri        = "/rest/fc-networks"
		q          map[string]interface{}
		fcNetworks FCNetworkList
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
		return fcNetworks, err
	}

	log.Debugf("GetfcNetworks %s", data)
	if err := json.Unmarshal([]byte(data), &fcNetworks); err != nil {
		return fcNetworks, err
	}
	return fcNetworks, nil
}

func (c *OVClient) CreateFCNetwork(fcNet FCNetwork) error {
	log.Infof("Initializing creation of fc network for %s.", fcNet.Name)
	var (
		uri = "/rest/fc-networks"
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, fcNet)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, fcNet)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new fc network request: %s", err)
		return err
	}

	log.Debugf("Response New fcNetwork %s", data)
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

func (c *OVClient) DeleteFCNetwork(name string) error {
	var (
		fcNet FCNetwork
		err   error
		t     *Task
		uri   string
	)

	fcNet, err = c.GetFCNetworkByName(name)
	if err != nil {
		return err
	}
	if fcNet.Name != "" {
		t = t.NewProfileTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", fcNet.URI, fcNet)
		log.Debugf("task -> %+v", t)
		uri = fcNet.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting new fc network request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete fc network %s", data)
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
		log.Infof("fcNetwork could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateFcNetwork(fcNet FCNetwork) error {
	log.Infof("Initializing update of fc network for %s.", fcNet.Name)
	var (
		uri = fcNet.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, fcNet)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, fcNet)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update fc network request: %s", err)
		return err
	}

	log.Debugf("Response Update FCNetwork %s", data)
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
