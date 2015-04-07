package module

import (
	"bytes"
	"encoding/gob"

	"github.com/hashicorp/terraform/config"
)

func (t *Tree) GobDecode(bs []byte) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Decode the gob data
	var data treeGob
	dec := gob.NewDecoder(bytes.NewReader(bs))
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

func (t *Tree) GobEncode() ([]byte, error) {
	data := &treeGob{
		Config:   t.config,
		Children: t.children,
		Name:     t.name,
		Path:     t.path,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// treeGob is used as a structure to Gob encode a tree.
//
// This structure is private so it can't be referenced but the fields are
// public, allowing Gob to properly encode this. When we decode this, we are
// able to turn it into a Tree.
type treeGob struct {
	Config   *config.Config
	Children map[string]*Tree
	Name     string
	Path     []string
}
