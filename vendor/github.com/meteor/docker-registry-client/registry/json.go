package registry

import (
	"encoding/json"
)

func (registry *Registry) getJson(url string, response interface{}) error {
	resp, err := registry.Client.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(response)
	if err != nil {
		return err
	}

	return nil
}
