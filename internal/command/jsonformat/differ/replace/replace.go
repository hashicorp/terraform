package replace

import "encoding/json"

// ForcesReplacement encapsulates the ReplacePaths logic from the Terraform
// change object.
//
// It is possible for a change to a deeply nested attribute or block to result
// in an entire resource being replaced (deleted then recreated) instead of
// simply updated. In this case, we want to attach some additional context to
// say, this resource is being replaced because of these changes to its
// internal values.
//
// The ReplacePaths field is a slice of paths that point to the values causing
// the replace operation. It's a slice of paths because you can have multiple
// internal values causing a replacement.
//
// Each path is a slice of indices, where an index can be a string or an
// integer. We represent this a slice of generic interfaces: []interface{}. This
// is because we actually parse this field from JSON and have no way to easily
// represent a value that can be a string or an integer in Go. Luckily, this
// doesn't matter too much from an implementation point of view because we
// always know what type to expect as we know whether we are currently looking
// at a list type (which means an integer) or a map type (which means a string).
//
// The GetChildWithKey and GetChildWithIndex return additional but modified
// ForcesReplacement objects, where a path is simply dropped if the index
// doesn't match or included with the first entry removed if the index did
// match. These functions are called as the outside Change objects are being
// created for a complex change's children.
//
// The ForcesReplacement function actually tells you whether the current value
// is causing a replacement operation as one of the paths will be empty since
// we removed an entry every time the path matched, and the last entry will have
// been removed when the change was created.
type ForcesReplacement struct {
	ReplacePaths [][]interface{}
}

// Parse accepts a json.RawMessage and outputs a formatted ForcesReplacement
// object.
//
// Parse expects the message to be a JSON array of JSON arrays containing
// strings and floats. This function happily accepts a null input representing
// none of the changes in this resource are causing a replacement.
func Parse(message json.RawMessage) ForcesReplacement {
	replace := ForcesReplacement{}
	if message == nil {
		return replace
	}

	if err := json.Unmarshal(message, &replace.ReplacePaths); err != nil {
		panic("failed to unmarshal replace paths: " + err.Error())
	}

	return replace
}

// ForcesReplacement returns true if this ForcesReplacement object represents
// a change that is causing the entire resource to be replaced.
func (replace ForcesReplacement) ForcesReplacement() bool {
	for _, path := range replace.ReplacePaths {
		if len(path) == 0 {
			return true
		}
	}
	return false
}

// GetChildWithKey steps through the paths in this ForcesReplacement and checks
// if any match the specified key.
//
// This function assumes the index will all be strings, so callers have to be
// sure they have navigated through previous paths accurately to this point or
// this function is liable to panic.
func (replace ForcesReplacement) GetChildWithKey(key string) ForcesReplacement {
	child := ForcesReplacement{}
	for _, path := range replace.ReplacePaths {
		if len(path) == 0 {
			// This means that the current value is causing a replacement but
			// not its children, so we skip as we are returning the child's
			// value.
			continue
		}

		if path[0].(string) == key {
			child.ReplacePaths = append(child.ReplacePaths, path[1:])
		}
	}
	return child
}

// GetChildWithIndex steps through the paths in this ForcesReplacement and
// checks if any match the specified index.
//
// This function assumes the index will all be integers, so callers have to be
// sure they have navigated through previous paths accurately to this point or
// this function is liable to panic.
func (replace ForcesReplacement) GetChildWithIndex(index int) ForcesReplacement {
	child := ForcesReplacement{}
	for _, path := range replace.ReplacePaths {
		if len(path) == 0 {
			// This means that the current value is causing a replacement but
			// not its children, so we skip as we are returning the child's
			// value.
			continue
		}

		if int(path[0].(float64)) == index {
			child.ReplacePaths = append(child.ReplacePaths, path[1:])
		}
	}
	return child
}
