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

type KickstartFile struct {
	Name string // The name the kickstart file will be saved in Cobbler
	Body string // The contents of the kickstart file
}

// Creates a kickstart file in Cobbler.
// Takes a KickstartFile struct as input.
// Returns true/false and error if creation failed.
func (c *Client) CreateKickstartFile(f KickstartFile) error {
	_, err := c.Call("read_or_write_kickstart_template", f.Name, false, f.Body, c.Token)
	return err
}

// Gets a kickstart file in Cobbler.
// Takes a kickstart file name as input.
// Returns *KickstartFile and error if read failed.
func (c *Client) GetKickstartFile(ksName string) (*KickstartFile, error) {
	result, err := c.Call("read_or_write_kickstart_template", ksName, true, "", c.Token)

	if err != nil {
		return nil, err
	}

	ks := KickstartFile{
		Name: ksName,
		Body: result.(string),
	}

	return &ks, nil
}

// Deletes a kickstart file in Cobbler.
// Takes a kickstart file name as input.
// Returns error if delete failed.
func (c *Client) DeleteKickstartFile(name string) error {
	_, err := c.Call("read_or_write_kickstart_template", name, false, -1, c.Token)
	return err
}
