package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type FCoENetwork struct {
	Type                  string        `json:"type,omitempty"`
	VlanId                int           `json:"vlanId,omitempty"`
	ConnectionTemplateUri utils.Nstring `json:"connectionTemplateUri,omitempty"`
	ManagedSanUri         utils.Nstring `json:"managedSanUri,omitempty"`
	FabricUri             utils.Nstring `json:"fabricUri,omitempty"`
	Description           utils.Nstring `json:"description,omitempty"`
	Name                  string        `json:"name,omitempty"`
	State                 string        `json:"state,omitempty"`
	Status                string        `json:"status,omitempty"`
	ETAG                  string        `json:"eTag,omitempty"`
	Modified              string        `json:"modified,omitempty"`
	Created               string        `json:"created,omitempty"`
	Category              string        `json:"category,omitempty"`
	URI                   utils.Nstring `json:"uri,omitempty"`
}

type FCoENetworkList struct {
	Total       int           `json:"total,omitempty"`       // "total": 1,
	Count       int           `json:"count,omitempty"`       // "count": 1,
	Start       int           `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []FCoENetwork `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetFCoENetworkByName(name string) (FCoENetwork, error) {
	var (
		fcoeNet FCoENetwork
	)
	fcoeNets, err := c.GetFCoENetworks(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if fcoeNets.Total > 0 {
		return fcoeNets.Members[0], err
	} else {
		return fcoeNet, err
	}
}

func (c *OVClient) GetFCoENetworks(filter string, sort string) (FCoENetworkList, error) {
	var (
		uri          = "/rest/fcoe-networks"
		q            map[string]interface{}
		fcoeNetworks FCoENetworkList
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
		return fcoeNetworks, err
	}

	log.Debugf("GetfcoeNetworks %s", data)
	if err := json.Unmarshal([]byte(data), &fcoeNetworks); err != nil {
		return fcoeNetworks, err
	}
	return fcoeNetworks, nil
}

func (c *OVClient) CreateFCoENetwork(fcoeNet FCoENetwork) error {
	log.Infof("Initializing creation of fcoe network for %s.", fcoeNet.Name)
	var (
		uri = "/rest/fcoe-networks"
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, fcoeNet)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, fcoeNet)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new fcoe network request: %s", err)
		return err
	}

	log.Debugf("Response New fcoeNetwork %s", data)
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

func (c *OVClient) DeleteFCoENetwork(name string) error {
	var (
		fcoeNet FCoENetwork
		err     error
		t       *Task
		uri     string
	)

	fcoeNet, err = c.GetFCoENetworkByName(name)
	if err != nil {
		return err
	}
	if fcoeNet.Name != "" {
		t = t.NewProfileTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", fcoeNet.URI, fcoeNet)
		log.Debugf("task -> %+v", t)
		uri = fcoeNet.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting deleting fcoe network request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete fcoe network %s", data)
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
		log.Infof("fcoeNetwork could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateFCoENetwork(fcoeNet FCoENetwork) error {
	log.Infof("Initializing update of fcoe network for %s.", fcoeNet.Name)
	var (
		uri = fcoeNet.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, fcoeNet)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, fcoeNet)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update fcoe network request: %s", err)
		return err
	}

	log.Debugf("Response Update FCoENetwork %s", data)
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
