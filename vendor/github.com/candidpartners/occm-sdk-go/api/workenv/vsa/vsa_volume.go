// Package implements OCCM Working Environments API (VSA)
package vsa

import (
  "encoding/json"

  "github.com/candidpartners/occm-sdk-go/api/client"
  "github.com/candidpartners/occm-sdk-go/api/workenv"
  "github.com/pkg/errors"
)

// VSA volume create request
type VSAVolumeCreateRequest struct {
  WorkingEnvironmentId    string                              `json:"workingEnvironmentId,omitempty"`
  SvmName                 string                              `json:"svmName,omitempty"`
  AggregateName           string                              `json:"aggregateName,omitempty"`
  Name                    string                              `json:"name,omitempty"`
  Size                    *workenv.Capacity                   `json:"size,omitempty"`
  InitialSize             *workenv.Capacity                   `json:"initialSize,omitempty"`
  SnapshotPolicyName      string                              `json:"snapshotPolicyName,omitempty"`
  ExportPolicyInfo        *workenv.ExportPolicyInfo           `json:"exportPolicyInfo,omitempty"`
  ShareInfo               *workenv.CreateCIFSShareInfoRequest `json:"shareInfo,omitempty"`
  ThinProvisioning        bool                                `json:"enableThinProvisioning"`
  Compression             bool                                `json:"enableCompression"`
  Deduplication           bool                                `json:"enableDeduplication"`
  CapacityTier            string                              `json:"capacityTier,omitempty"`
  ProviderVolumeType      string                              `json:"providerVolumeType,omitempty"`
  MaxNumOfDisksApprovedToAdd  int                             `json:"maxNumOfDisksApprovedToAdd"`
  SyncToS3                bool                                `json:"syncToS3,omitempty"`
  IOPS                    int                                 `json:"iops,omitempty"`
}

// VSA volume quote request
type VSAVolumeQuoteRequest struct {
  WorkingEnvironmentId    string                              `json:"workingEnvironmentId,omitempty"`
  SvmName                 string                              `json:"svmName,omitempty"`
  AggregateName           string                              `json:"aggregateName,omitempty"`
  Name                    string                              `json:"name,omitempty"`
  Size                    *workenv.Capacity                   `json:"size,omitempty"`
  InitialSize             *workenv.Capacity                   `json:"initialSize,omitempty"`
  ThinProvisioning        bool                                `json:"enableThinProvisioning"`
  CapacityTier            string                              `json:"capacityTier,omitempty"`
  ProviderVolumeType      string                              `json:"providerVolumeType,omitempty"`
  VerifyNameUniqueness    bool                                `json:"verifyNameUniqueness"`
  IOPS                    int                                 `json:"iops,omitempty"`
}

type VSAVolumeQuoteResponse struct {
  NumOfDisks                int                               `json:"numOfDisks"`
  DiskSize                  *workenv.Capacity                 `json:"diskSize,omitempty"`
  AggregateName             string                            `json:"aggregateName,omitempty"`
  NewAggregate              bool                              `json:"newAggregate"`
  AutoVSACapacityManagement bool                              `json:"autoVsaCapacityManagement"`
}

func VolumeQuoteResponseFromJSON(data []byte) (*VSAVolumeQuoteResponse, error) {
  var result VSAVolumeQuoteResponse
  err := json.Unmarshal(data, &result)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return &result, nil
}
