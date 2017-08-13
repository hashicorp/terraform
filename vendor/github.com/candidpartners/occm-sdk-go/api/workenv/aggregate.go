// Package implements OCCM Working Environments API
package workenv

import (
  "encoding/json"

  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Aggregate response wrapper object
type AggregateResponse struct {
  Name              string    `json:"name"`
  AvailableCapacity *Capacity `json:"availableCapacity"`
  TotalCapacity     *Capacity `json:"totalCapacity"`
  Volumes           []Volume  `json:"volumes"`
  ProviderVolumes   []ProviderVolume  `json:"providerVolumes"`
  Disks             []Disk    `json:"disks"`
  State             string    `json:"state"`
  EncryptionType    string    `json:"encryptionType"`
  EncryptionKeyId   string    `json:"encryptionKeyId"`
  HomeNode          string    `json:"homeNode"`
  OwnerNode         string    `json:"ownerNode"`
  CapacityTier      string    `json:"capacityTier"`
  Root              bool      `json:"root"`
}

// TODO: add on-prem aggregate

func AggregateResponseListFromJSON(data []byte) ([]AggregateResponse, error) {
  var result []AggregateResponse
  err := json.Unmarshal(data, &result)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return result, nil
}
