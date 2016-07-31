/*
Copyright 2015 Container Solutions

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cobblerclient

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/kolo/xmlrpc"
	"github.com/mitchellh/mapstructure"
)

const bodyTypeXML = "text/xml"

type HTTPClient interface {
	Post(string, string, io.Reader) (*http.Response, error)
}

type Client struct {
	httpClient HTTPClient
	config     ClientConfig
	Token      string
}

type ClientConfig struct {
	Url      string
	Username string
	Password string
}

func NewClient(httpClient HTTPClient, c ClientConfig) Client {
	return Client{
		httpClient: httpClient,
		config:     c,
	}
}

func (c *Client) Call(method string, args ...interface{}) (interface{}, error) {
	var result interface{}

	reqBody, err := xmlrpc.EncodeMethodCall(method, args...)
	if err != nil {
		return nil, err
	}

	r := fmt.Sprintf("%s\n", string(reqBody))
	res, err := c.httpClient.Post(c.config.Url, bodyTypeXML, bytes.NewReader([]byte(r)))
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	resp := xmlrpc.NewResponse(body)
	if err := resp.Unmarshal(&result); err != nil {
		return nil, err
	}

	if resp.Failed() {
		return nil, resp.Err()
	}

	return result, nil
}

// Performs a login request to Cobbler using the credentials provided
// in the configuration in the initializer.
func (c *Client) Login() (bool, error) {
	result, err := c.Call("login", c.config.Username, c.config.Password)
	if err != nil {
		return false, err
	}

	c.Token = result.(string)
	return true, nil
}

// Sync the system.
// Returns an error if anything went wrong
func (c *Client) Sync() error {
	_, err := c.Call("sync", c.Token)
	return err
}

// GetItemHandle gets the internal ID of a Cobbler item.
func (c *Client) GetItemHandle(what, name string) (string, error) {
	result, err := c.Call("get_item_handle", what, name, c.Token)
	if err != nil {
		return "", err
	} else {
		return result.(string), err
	}
}

// cobblerDataHacks is a hook for the mapstructure decoder. It's only used by
// decodeCobblerItem and should never be invoked directly.
// It's used to smooth out issues with converting fields and types from Cobbler.
func cobblerDataHacks(f, t reflect.Kind, data interface{}) (interface{}, error) {
	dataVal := reflect.ValueOf(data)

	// Cobbler uses ~ internally to mean None/nil
	if dataVal.String() == "~" {
		return map[string]interface{}{}, nil
	}

	if f == reflect.Int64 && t == reflect.Bool {
		if dataVal.Int() > 0 {
			return true, nil
		} else {
			return false, nil
		}
	}
	return data, nil
}

// decodeCobblerItem is a custom mapstructure decoder to handler Cobbler's uniqueness.
func decodeCobblerItem(raw interface{}, result interface{}) (interface{}, error) {
	var metadata mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:         &metadata,
		Result:           result,
		WeaklyTypedInput: true,
		DecodeHook:       cobblerDataHacks,
	})

	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(raw); err != nil {
		return nil, err
	}

	return result, nil
}

// updateCobblerFields updates all fields in a Cobbler Item structure.
func (c *Client) updateCobblerFields(what string, item reflect.Value, id string) error {
	method := fmt.Sprintf("modify_%s", what)

	typeOfT := item.Type()
	for i := 0; i < item.NumField(); i++ {
		v := item.Field(i)
		tag := typeOfT.Field(i).Tag
		field := tag.Get("mapstructure")
		cobblerTag := tag.Get("cobbler")

		if cobblerTag == "noupdate" {
			continue
		}

		if field == "" {
			continue
		}

		var value interface{}
		switch v.Type().String() {
		case "string", "bool", "int64", "int":
			value = v.Interface()
		case "[]string":
			value = strings.Join(v.Interface().([]string), " ")
		}

		//fmt.Printf("%s, %s, %s\n", id, field, value)
		if result, err := c.Call(method, id, field, value, c.Token); err != nil {
			return err
		} else {
			if result.(bool) == false && value != false {
				return fmt.Errorf("Error updating %s to %s.", field, value)
			}
		}
	}

	return nil
}
