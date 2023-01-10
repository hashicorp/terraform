package replace

import "encoding/json"

type ForcesReplacement struct {
	ReplacePaths [][]interface{}
}

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

func (replace ForcesReplacement) ForcesReplacement() bool {
	for _, path := range replace.ReplacePaths {
		if len(path) == 0 {
			return true
		}
	}
	return false
}

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
