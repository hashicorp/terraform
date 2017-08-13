// Package implements OCCM Tenant API
package tenant

import (
  "encoding/json"

  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Tenant object
type Tenant struct {
  Name        string `json:"name"`
  Description string `json:"description"`
  NssUser     string `json:"nssUserName"`
  PublicId    string `json:"publicId"`
  CostCenter  string `json:"costCenter"`
}

func ListFromJSON(data []byte) ([]Tenant, error) {
  var result []Tenant
  err := json.Unmarshal(data, &result)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return result, nil
}
