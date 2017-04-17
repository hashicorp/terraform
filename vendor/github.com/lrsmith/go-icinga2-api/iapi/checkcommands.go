package iapi

import (
	"encoding/json"
	"fmt"
)

// GetCheckcommand ...
func (server *Server) GetCheckcommand(name string) ([]CheckcommandStruct, error) {

	var checkcommands []CheckcommandStruct
	results, err := server.NewAPIRequest("GET", "/objects/checkcommands/"+name, nil)
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

	if unmarshalErr := json.Unmarshal(jsonStr, &checkcommands); unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return checkcommands, err

}

// CreateCheckcommand ...
func (server *Server) CreateCheckcommand(name, command string, command_arguments map[string]string) ([]CheckcommandStruct, error) {

	var newAttrs CheckcommandAttrs
	newAttrs.Command = []string{command}
	newAttrs.Arguments = command_arguments

	var newCheckcommand CheckcommandStruct
	newCheckcommand.Name = name
	newCheckcommand.Type = "CheckCommand"
	newCheckcommand.Attrs = newAttrs

	// Create JSON from completed struct
	payloadJSON, marshalErr := json.Marshal(newCheckcommand)
	if marshalErr != nil {
		return nil, marshalErr
	}

	//fmt.Printf("<payload> %s\n", payloadJSON)

	// Make the API request to create the hosts.
	results, err := server.NewAPIRequest("PUT", "/objects/checkcommands/"+name, []byte(payloadJSON))
	if err != nil {
		return nil, err
	}

	if results.Code == 200 {
		theCheckcommand, err := server.GetCheckcommand(name)
		return theCheckcommand, err
	}

	return nil, fmt.Errorf("%s", results.ErrorString)

}

// DeleteCheckcommand ...
func (server *Server) DeleteCheckcommand(name string) error {

	results, err := server.NewAPIRequest("DELETE", "/objects/checkcommands/"+name+"?cascade=1", nil)
	if err != nil {
		return err
	}

	if results.Code == 200 {
		return nil
	} else {
		return fmt.Errorf("%s", results.ErrorString)
	}

}
