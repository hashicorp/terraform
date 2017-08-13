// Utility package
package util

import (
  "bytes"
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

// FromJSONStream parses JSON from a stream
func FromJSONStream(r io.Reader) (interface{}, error) {
  b, err := ioutil.ReadAll(r)
  if err != nil {
    return nil, errors.Wrap(err, "Error reading data")
  }

  return FromJSON(b)
}

// FromJSON parses JSON from an array
func FromJSON(b []byte) (interface{}, error) {
  var result interface{}
  err := json.Unmarshal(b, &result)
  if err != nil {
    return nil, errors.Wrap(err, "Error parsing JSON")
  }

  return result, nil
}

// ToJSON converts an object to JSON stream
func ToJSONStream(i interface{}) (io.Reader, error) {
  json, err := json.Marshal(i)
  if err != nil {
    return nil, errors.Wrap(err, "Error parsing object")
  }
  return bytes.NewReader(json), nil
}

func ToString(v interface{}) string {
  b, _ := json.MarshalIndent(v, "", "  ")
  return string(b)
}

func GetRequestIdHeader(h map[string][]string) (string, error) {
  val := h["Oncloud-Request-Id"]
  if val != nil && len(val) > 0 {
    return val[0], nil
  }
  return "", errors.New("Missing Oncloud-Request-Id header")
}
