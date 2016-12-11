package module

import (
	"bytes"
	"encoding/json"

	"github.com/hashicorp/terraform/config"
)

func (t *Tree) UnmarshalJSON(bs []byte) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Decode the gob data
	var data treeJSON
	dec := json.NewDecoder(bytes.NewReader(bs))
	if err := dec.Decode(&data); err != nil {
		return err
	}

	// Set the fields
	t.name = data.Name
	t.config = data.Config
	t.children = data.Children
	t.path = data.Path

	return nil
}

func (t *Tree) MarshalJSON() ([]byte, error) {
	data := &treeJSON{
		Config:   t.config,
		Children: t.children,
		Name:     t.name,
		Path:     t.path,
	}

	return json.Marshal(data)
}

// treeJSON is used as a structure to JSON encode a tree.
//
// This structure is private so it can't be referenced but the fields are
// public, allowing us to properly encode this. When we decode this, we are
// able to turn it into a Tree.
type treeJSON struct {
	Config   *config.Config   `json:"config"`
	Children map[string]*Tree `json:"children"`
	Name     string           `json:"name"`
	Path     []string         `json:"path"`
}
