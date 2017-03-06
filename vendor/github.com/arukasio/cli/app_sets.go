package arukas

import (
	"encoding/json"
	"github.com/manyminds/api2go/jsonapi"
)

// TmpJSON Contain JSON data.
type TmpJSON struct {
	Data []map[string]interface{} `json:"data"`
	Meta map[string]interface{}   `json:"-"`
}

// AppSet represents a application data in struct variables.
type AppSet struct {
	App       App
	Container Container
}

// MarshalJSON returns as as the JSON encoding of as.
func (as AppSet) MarshalJSON() ([]byte, error) {
	var (
		app           []byte
		appJSON       map[string]map[string]interface{}
		container     []byte
		containerJSON map[string]map[string]interface{}
		marshaled     []byte
		err           error
	)

	if app, err = jsonapi.Marshal(as.App); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(app, &appJSON); err != nil {
		return nil, err
	}

	if container, err = jsonapi.Marshal(as.Container); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(container, &containerJSON); err != nil {
		return nil, err
	}

	data := map[string][]map[string]interface{}{
		"data": []map[string]interface{}{
			appJSON["data"],
			containerJSON["data"],
		},
	}

	if marshaled, err = json.Marshal(data); err != nil {
		return nil, err
	}

	return marshaled, nil
}

// SelectResources returns the type filter value of TmpJSON.
func SelectResources(data TmpJSON, resourceType string) map[string][]map[string]interface{} {
	var resources []map[string]interface{}
	// resources := make([]map[string]interface{}, 0)

	for _, v := range data.Data {
		if v["type"] == resourceType {
			resources = append(resources, v)
		}
	}

	filtered := map[string][]map[string]interface{}{
		"data": resources,
	}
	return filtered
}

// UnmarshalJSON sets *as to a copy of data.
func (as *AppSet) UnmarshalJSON(bytes []byte) error {
	var (
		appBytes       []byte
		containerBytes []byte
		err            error
		data           TmpJSON
	)
	if err = json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	apps := SelectResources(data, "apps")
	containers := SelectResources(data, "containers")

	if appBytes, err = json.Marshal(apps); err != nil {
		return err
	}

	if containerBytes, err = json.Marshal(containers); err != nil {
		return err
	}

	var parsedApps []App
	if err = jsonapi.Unmarshal(appBytes, &parsedApps); err != nil {
		return err
	}

	var parsedContainers []Container
	if err = jsonapi.Unmarshal(containerBytes, &parsedContainers); err != nil {
		return err
	}

	as.App = parsedApps[0]
	as.Container = parsedContainers[0]
	return nil
}
