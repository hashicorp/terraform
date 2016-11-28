package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type StorageVolumeV3 struct {
	Category               string                 `json:"category,omitempty"`
	Created                string                 `json:"created,omitempty"`
	Description            string                 `json:"description,omitempty"`
	ETAG                   string                 `json:"eTag,omitempty"`
	Name                   string                 `json:"name,omitempty"`
	State                  string                 `json:"state,omitempty"`
	Status                 string                 `json:"status,omitempty"`
	Type                   string                 `json:"type,omitempty"`
	URI                    utils.Nstring          `json:"uri,omitempty"`
	Shareable              bool                   `json:"shareable,omitempty"`
	StoragePoolUri         utils.Nstring          `json:"storagePoolUri,omitempty"`
	StorageSystemUri       utils.Nstring          `json:"storageSystemUri,omitempty"`
	ProvisionType          string                 `json:"provisionType,omitempty"`
	ProvisionedCapacity    string                 `json:"provisionedCapacity,omitempty"`
	ProvisioningParameters ProvisioningParameters `json:"provisioningParameters,omitempty"`
	//	Wwn										string				`json:""`

	/*
	   Category              string        `json:"category,omitempty"`              //  "type": "StorageVolumeV3",
	   Created               string        `json:"created,omitempty"`               //  "created": "2016-06-20T22:48:14.422Z"
	   Description           string        `json:"description,omitempty"`           //  "description": "Integration test volume 2",
	   ETAG                  string        `json:"eTag,omitempty"`                  //  "eTag": "2016-06-20T22:48:17.704Z",            string        `json:"modified,omitempty"`              // "modified": "20150831T154835.250Z",
	   Name                  string        `json:"name,omitempty"`                  //  "name": "Volume_2",
	   State                 string        `json:"state,omitempty"`                 //  "state": "Configured",
	   Status                string        `json:"status,omitempty"`                //  "status": "OK",
	   Type                  string        `json:"type,omitempty"`                  //  "type": "StorageVolumeV3",
	   URI                   utils.Nstring `json:"uri,omitempty"`                   //  "uri": "/rest/storage-volumes/527801AC-B6B6-4A63-8510-D32906C9C57B"
	   Shareable							string				`json:"shareable,omitempty"`                                //  "shareable": false,
	   AllocatedCapacity			string				`json:""`																 //  "allocatedCapacity": "1073741824",
	   DeviceType						string				`json:""`																 //  "deviceType": "SSD",
	   DeviceVolumeName			string				`json:""`                                //  "deviceVolumeName": "Volume_2",
	   IsPermanent						string 				`json:""`                                //  "isPermanent": true,
	   RaidLevel							string				`json:""`                								 //  "raidLevel": "RAID5",
	   RefreshState					string				`json:""`                                //  "refreshState": "NotRefreshing",
	   RevertToSnapshotUri		string				`json:""`                                //  "revertToSnapshotUri": null,
	   SnapshotPoolUri				string				`json:""`                                //  "snapshotPoolUri": "/rest/storage-pools/000D692C-74B3-44AC-B297-3741AABE29F8",
	   Snapshots							string				`json:""`                                //  "snapshots": null,
	   StateReason						string				`json:""`                                //  "stateReason": "None",
	   StoragePoolUri				string				`json:"storagePoolUri,omitempty"`        //  "storagePoolUri": "/rest/storage-pools/000D692C-74B3-44AC-B297-3741AABE29F8",
	   StorageSystemUri			string				`json:"storageSystemUri,omitempty"`      //  "storageSystemUri": "/rest/storage-systems/TXQ1000307",
	   ProvisionType					string 				`json:"provisionType,omitempty"`         //  "provisionType": "Full",
	   ProvisionedCapacity		string				`json:"provisionedCapacity,omitempty"`   //  "provisionedCapacity": "1073741824",
	   ProvisioningParameters []ProvisioningParameters		`json:"provisioningParameters,omitempty"`
	   //	Wwn										string				`json:""`                                //  "wwn": "DC:57:93:56:32:00:10:00:30:71:46:64:62:89:52:11",


	*/
}

type ProvisioningParameters struct {
	StoragePoolUri    utils.Nstring `json:"storagePoolUri,omitempty"`
	ProvisionType     string        `json:"provisionType,omitempty"`
	RequestedCapacity string        `json:"requestedCapacity,omitempty"`
	Shareable         bool          `json:"shareable,omitempty"`
}

type StorageVolumesListV3 struct {
	Total       int               `json:"total,omitempty"`       // "total": 1,
	Count       int               `json:"count,omitempty"`       // "count": 1,
	Start       int               `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring     `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring     `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring     `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []StorageVolumeV3 `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetStorageVolumeByName(name string) (StorageVolumeV3, error) {
	var (
		sVol StorageVolumeV3
	)
	sVols, err := c.GetStorageVolumes(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if sVols.Total > 0 {
		return sVols.Members[0], err
	} else {
		return sVol, err
	}
}

func (c *OVClient) GetStorageVolumes(filter string, sort string) (StorageVolumesListV3, error) {
	var (
		uri   = "/rest/storage-volumes"
		q     map[string]interface{}
		sVols StorageVolumesListV3
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
		return sVols, err
	}

	log.Debugf("GetStorageVolumes %s", data)
	if err := json.Unmarshal([]byte(data), &sVols); err != nil {
		return sVols, err
	}
	return sVols, nil
}

func (c *OVClient) CreateStorageVolume(sVol StorageVolumeV3) error {
	log.Infof("Initializing creation of storage volume for %s.", sVol.Name)
	var (
		uri = "/rest/storage-volumes"
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, sVol)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, sVol)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new storage volume request: %s", err)
		return err
	}

	log.Debugf("Response New StorageVolume %s", data)
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

func (c *OVClient) DeleteStorageVolume(name string) error {
	var (
		sVol StorageVolumeV3
		err  error
		t    *Task
		uri  string
	)

	sVol, err = c.GetStorageVolumeByName(name)
	if err != nil {
		return err
	}
	if sVol.Name != "" {
		t = t.NewProfileTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", sVol.URI, sVol)
		log.Debugf("task -> %+v", t)
		uri = sVol.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete storage volume request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete storage volume %s", data)
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
		log.Infof("StorageVolume could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateStorageVolume(sVol StorageVolumeV3) error {
	log.Infof("Initializing update of storage volume for %s.", sVol.Name)
	var (
		uri = sVol.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, sVol)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, sVol)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update StorageVolume request: %s", err)
		return err
	}

	log.Debugf("Response update StorageVolume %s", data)
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
