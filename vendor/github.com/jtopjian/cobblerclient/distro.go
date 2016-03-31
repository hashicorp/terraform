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
	"fmt"
	"reflect"
)

// Distro is a created distro.
type Distro struct {
	// These are internal fields and cannot be modified.
	Ctime         float64 `mapstructure:"ctime"          cobbler:"noupdate"` // TODO: convert to time
	Depth         int     `mapstructure:"depth"          cobbler:"noupdate"`
	ID            string  `mapstructure:"uid"            cobbler:"noupdate"`
	Mtime         float64 `mapstructure:"mtime"          cobbler:"noupdate"` // TODO: convert to time
	TreeBuildTime string  `mapstructure:tree_build_time" cobbler:"noupdate"`

	Arch                   string   `mapstructure:"arch"`
	Breed                  string   `mapstructure:"breed"`
	BootFiles              string   `mapstructure:"boot_files"`
	Comment                string   `mapstructure:"comment"`
	FetchableFiles         string   `mapstructure:"fetchable_files"`
	Kernel                 string   `mapstructure:"kernel"`
	KernelOptions          string   `mapstructure:"kernel_options"`
	KernelOptionsPost      string   `mapstructure:"kernel_options_post"`
	Initrd                 string   `mapstructure:"initrd"`
	MGMTClasses            []string `mapstructure:"mgmt_classes"`
	Name                   string   `mapstructure:"name"`
	OSVersion              string   `mapstructure:"os_version"`
	Owners                 []string `mapstructure:"owners"`
	RedHatManagementKey    string   `mapstructure:"redhat_management_key"`
	RedHatManagementServer string   `mapstructure:"redhat_management_server"`
	TemplateFiles          string   `mapstructure:"template_files"`

	//KSMeta                 string   `mapstructure:"ks_meta"`
	//SourceRepos            []string `mapstructure:"source_repos"`
}

// GetDistros returns all systems in Cobbler.
func (c *Client) GetDistros() ([]*Distro, error) {
	var distros []*Distro

	result, err := c.Call("get_distros", "-1", c.Token)
	if err != nil {
		return nil, err
	}

	for _, d := range result.([]interface{}) {
		var distro Distro
		decodedResult, err := decodeCobblerItem(d, &distro)
		if err != nil {
			return nil, err
		}

		distros = append(distros, decodedResult.(*Distro))
	}

	return distros, nil
}

// GetDistro returns a single distro obtained by its name.
func (c *Client) GetDistro(name string) (*Distro, error) {
	var distro Distro

	result, err := c.Call("get_distro", name, c.Token)
	if result == "~" {
		return nil, fmt.Errorf("Distro %s not found.", name)
	}

	if err != nil {
		return nil, err
	}

	decodeResult, err := decodeCobblerItem(result, &distro)
	if err != nil {
		return nil, err
	}

	return decodeResult.(*Distro), nil
}

// CreateDistro creates a distro.
func (c *Client) CreateDistro(distro Distro) (*Distro, error) {
	// Make sure a distro with the same name does not already exist
	if _, err := c.GetDistro(distro.Name); err == nil {
		return nil, fmt.Errorf("A Distro with the name %s already exists.", distro.Name)
	}

	result, err := c.Call("new_distro", c.Token)
	if err != nil {
		return nil, err
	}
	newId := result.(string)

	item := reflect.ValueOf(&distro).Elem()
	if err := c.updateCobblerFields("distro", item, newId); err != nil {
		return nil, err
	}

	if _, err := c.Call("save_distro", newId, c.Token); err != nil {
		return nil, err
	}

	return c.GetDistro(distro.Name)
}

// UpdateDistro updates a single distro.
func (c *Client) UpdateDistro(distro *Distro) error {
	item := reflect.ValueOf(distro).Elem()
	id, err := c.GetItemHandle("distro", distro.Name)
	if err != nil {
		return err
	}

	if err := c.updateCobblerFields("distro", item, id); err != nil {
		return err
	}

	if _, err := c.Call("save_distro", id, c.Token); err != nil {
		return err
	}

	return nil
}

// DeleteDistro deletes a single distro by its name.
func (c *Client) DeleteDistro(name string) error {
	_, err := c.Call("remove_distro", name, c.Token)
	return err
}
