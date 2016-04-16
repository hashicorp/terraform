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

type Snippet struct {
	Name string // The name the snippet file will be saved in Cobbler
	Body string // The contents of the kickstart file
}

// Creates a snippet in Cobbler.
// Takes a Snippet struct as input
// Returns true/false and error if creation failed.
func (c *Client) CreateSnippet(s Snippet) error {
	_, err := c.Call("read_or_write_snippet", s.Name, false, s.Body, c.Token)
	return err
}

// Gets a snippet file in Cobbler.
// Takes a snippet file name as input.
// Returns *Snippet and error if read failed.
func (c *Client) GetSnippet(name string) (*Snippet, error) {
	result, err := c.Call("read_or_write_snippet", name, true, "", c.Token)

	if err != nil {
		return nil, err
	}

	snippet := Snippet{
		Name: name,
		Body: result.(string),
	}

	return &snippet, nil
}

// Gets a snippet file in Cobbler.
// Takes a snippet file name as input.
// Returns error if delete failed.
func (c *Client) DeleteSnippet(name string) error {
	_, err := c.Call("read_or_write_snippet", name, false, -1, c.Token)
	return err
}
