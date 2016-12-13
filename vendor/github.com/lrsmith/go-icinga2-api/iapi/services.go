package iapi

import (
	"encoding/json"
	"fmt"
)

// GetService ...
func (server *Server) GetService(servicename, hostname string) ([]ServiceStruct, error) {

	var services []ServiceStruct
	results, err := server.NewAPIRequest("GET", "/objects/services/"+hostname+"!"+servicename, nil)
	if err != nil {
		return nil, err
	}

	// Contents of the results is an interface object. Need to convert it to json first.
	jsonStr, marshalErr := json.Marshal(results.Results)
	if marshalErr != nil {
		return nil, marshalErr
	}

	// then the JSON can be pushed into the appropriate struct.
	// Note : Results is a slice so much push into a slice.

	if unmarshalErr := json.Unmarshal(jsonStr, &services); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return services, err

}

// CreateService ...
func (server *Server) CreateService(servicename, hostname, checkCommand string) ([]ServiceStruct, error) {

	var newAttrs ServiceAttrs
	newAttrs.CheckCommand = checkCommand

	var newService ServiceStruct
	newService.Attrs = newAttrs

	// Create JSON from completed struct
	payloadJSON, marshalErr := json.Marshal(newService)
	if marshalErr != nil {
		return nil, marshalErr
	}

	//fmt.Printf("<payload> %s\n", payloadJSON)

	// Make the API request to create the hosts.
	results, err := server.NewAPIRequest("PUT", "/objects/services/"+hostname+"!"+servicename, []byte(payloadJSON))
	if err != nil {
		return nil, err
	}

	if results.Code == 200 {
		services, err := server.GetService(servicename, hostname)
		return services, err
	}

	return nil, fmt.Errorf("%s", results.ErrorString)

}

// DeleteService ...
func (server *Server) DeleteService(servicename, hostname string) error {

	results, err := server.NewAPIRequest("DELETE", "/objects/services/"+hostname+"!"+servicename+"?cascade=1", nil)
	if err != nil {
		return err
	}

	if results.Code == 200 {
		return nil
	} else {
		return fmt.Errorf("%s", results.ErrorString)
	}

}
