package internal

import "encoding/json"

// Labels is used for connect JSON formatting.
type Labels map[string]string

// MarshalJSON requires a comment for golint?
func (l Labels) MarshalJSON() ([]byte, error) {
	ls := make([]struct {
		Key   string `json:"label_type"`
		Value string `json:"label_value"`
	}, len(l))

	i := 0
	for key, val := range l {
		ls[i].Key = key
		ls[i].Value = val
		i++
	}

	return json.Marshal(ls)
}
