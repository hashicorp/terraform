package attribute_path

import "encoding/json"

// Matcher provides an interface for stepping through changes following an
// attribute path.
//
// GetChildWithKey and GetChildWithIndex will check if any of the internal paths
// match the provided key or index, and return a new Matcher that will match
// that children or potentially it's children.
//
// The caller of the above functions is required to know whether the next value
// in the path is a list type or an object type and call the relevant function,
// otherwise these functions will crash/panic.
//
// The Matches function returns true if the paths you have traversed until now
// ends.
type Matcher interface {

	// Matches returns true if we have reached the end of a path and found an
	// exact match.
	Matches() bool

	// MatchesPartial returns true if the current attribute is part of a path
	// but not necessarily at the end of the path.
	MatchesPartial() bool

	GetChildWithKey(key string) Matcher
	GetChildWithIndex(index int) Matcher
}

// Parse accepts a json.RawMessage and outputs a formatted Matcher object.
//
// Parse expects the message to be a JSON array of JSON arrays containing
// strings and floats. This function happily accepts a null input representing
// none of the changes in this resource are causing a replacement. The propagate
// argument tells the matcher to propagate any matches to the matched attributes
// children.
//
// In general, this function is designed to accept messages that have been
// produced by the lossy cty.Paths conversion functions within the jsonplan
// package. There is nothing particularly special about that conversion process
// though, it just produces the nested JSON arrays described above.
func Parse(message json.RawMessage, propagate bool) Matcher {
	matcher := &PathMatcher{
		Propagate: propagate,
	}
	if message == nil {
		return matcher
	}

	if err := json.Unmarshal(message, &matcher.Paths); err != nil {
		panic("failed to unmarshal attribute paths: " + err.Error())
	}

	return matcher
}

// PathMatcher is only visible to aid with testing, it is generally safer to
// use the Parse function as the entry point.
type PathMatcher struct {
	// We represent our internal paths as a [][]interface{} as the cty.Paths
	// conversion process is lossy. Since the type information is lost there
	// is no (easy) way to reproduce the original cty.Paths object. Instead,
	// we simply rely on the external callers to know the type information and
	// call the correct GetChild function.
	Paths [][]interface{}

	// Propagate tells the matcher that it should propagate any matches it finds
	// onto the children of that match.
	Propagate bool
}

func (p *PathMatcher) Matches() bool {
	for _, path := range p.Paths {
		if len(path) == 0 {
			return true
		}
	}
	return false
}

func (p *PathMatcher) MatchesPartial() bool {
	return len(p.Paths) > 0
}

func (p *PathMatcher) GetChildWithKey(key string) Matcher {
	child := &PathMatcher{
		Propagate: p.Propagate,
	}
	for _, path := range p.Paths {
		if len(path) == 0 {
			// This means that the current value matched, but not necessarily
			// it's child.

			if p.Propagate {
				// If propagate is true, then our child match our matches
				child.Paths = append(child.Paths, path)
			}

			// If not we would simply drop this path from our set of paths but
			// either way we just continue.
			continue
		}

		if path[0].(string) == key {
			child.Paths = append(child.Paths, path[1:])
		}
	}
	return child
}

func (p *PathMatcher) GetChildWithIndex(index int) Matcher {
	child := &PathMatcher{
		Propagate: p.Propagate,
	}
	for _, path := range p.Paths {
		if len(path) == 0 {
			// This means that the current value matched, but not necessarily
			// it's child.

			if p.Propagate {
				// If propagate is true, then our child match our matches
				child.Paths = append(child.Paths, path)
			}

			// If not we would simply drop this path from our set of paths but
			// either way we just continue.
			continue
		}

		if int(path[0].(float64)) == index {
			child.Paths = append(child.Paths, path[1:])
		}
	}
	return child
}

// AlwaysMatcher returns a matcher that will always match all paths.
func AlwaysMatcher() Matcher {
	return &alwaysMatcher{}
}

type alwaysMatcher struct{}

func (a *alwaysMatcher) Matches() bool {
	return true
}

func (a *alwaysMatcher) MatchesPartial() bool {
	return true
}

func (a *alwaysMatcher) GetChildWithKey(_ string) Matcher {
	return a
}

func (a *alwaysMatcher) GetChildWithIndex(_ int) Matcher {
	return a
}
