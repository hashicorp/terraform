// Package implements OCCM Working Environments API
package workenv

import (
  "encoding/json"

  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Working Environments wrapper object
// TODO: not a complete list of properties
type WorkingEnvironments struct {
  VSA     []VsaWorkingEnvironment `json:"vsaWorkingEnvironments"`
  OnPrem  []OnPremWorkingEnvironment `json:"onPremWorkingEnvironments"`
  Azure   []AzureWorkingEnvironment `json:"azureWorkingEnvironments"`
}

// VSA Working Environment object
// TODO: not a complete list of properties
type VsaWorkingEnvironment struct {
  PublicId  string `json:"publicId"`
  Name      string `json:"name"`
  TenantId  string `json:"tenantId"`
  SvmName   string `json:"svmName"`
  IsHA      bool   `json:"isHA"`
}

// OnPrem Working Environment object
// TODO: not a complete list of properties
type OnPremWorkingEnvironment struct {
  PublicId  string `json:"publicId"`
  Name      string `json:"name"`
  TenantId  string `json:"tenantId"`
}

// Azure Working Environment object
// TODO: not a complete list of properties
type AzureWorkingEnvironment struct {
  PublicId  string `json:"publicId"`
  Name      string `json:"name"`
  TenantId  string `json:"tenantId"`
  SvmName   string `json:"svmName"`
}

func ListFromJSON(data []byte) (*WorkingEnvironments, error) {
  var result WorkingEnvironments
  err := json.Unmarshal(data, &result)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return &result, nil
}
