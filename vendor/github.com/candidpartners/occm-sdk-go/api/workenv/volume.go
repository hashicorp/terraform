// Package implements OCCM Working Environments API
package workenv

import (
  "encoding/json"

  "github.com/candidpartners/occm-sdk-go/api/client"
  "github.com/pkg/errors"
)

// Capacity wrapper object
type Capacity struct {
  Size float64 `json:"size,omitempty"`
  Unit string `json:"unit,omitempty"`
}

// Volume wrapper object
type Volume struct {
  Name            string  `json:"name,omitempty"`
  TotalSize       *Capacity  `json:"totalSize,omitempty"`
  UsedSize        *Capacity  `json:"usedSize,omitempty"`
  ThinProvisioned bool    `json:"thinProvisioned"`
  RootVolume      bool    `json:"rootVolume"`
}

// Provider Volume wrapper object
type ProviderVolume struct {
  Id              string  `json:"id,omitempty"`
  Name            string  `json:"name,omitempty"`
  Size            *Capacity  `json:"size,omitempty"`
  State           string  `json:"state,omitempty"`
  Device          string `json:"device,omitempty"`
  InstanceId      string `json:"instanceId,omitempty"`
  DiskType        string `json:"diskType,omitempty"`
  Encryped        bool   `json:"encrypted"`
}

// Disk wrapper object
type Disk struct {
  Name            string  `json:"name,omitempty"`
  Position        string  `json:"position,omitempty"`
  OwnerNode       string  `json:"ownerNode,omitempty"`
  Device          string `json:"device,omitempty"`
}

// Export policy info wrapper
type ExportPolicyInfo struct {
  PolicyType      string    `json:"policyType,omitempty"`
  IPs             []string  `json:"ips"`
}

// Named export policy info wrapper
type NamedExportPolicyInfo struct {
  Named           string    `json:"name,omitempty"`
  PolicyType      string    `json:"policyType,omitempty"`
  IPs             []string  `json:"ips,omitempty"`
}

// CIFS user permission wrapper
type CIFSShareUserPermissions struct {
  Permission      string    `json:"permission,omitempty"`
  Users           []string  `json:"users,omitempty"`
}

// CIFS share info wrapper
type CIFSShareInfo struct {
  ShareName           string                      `json:"shareName,omitempty"`
  AccessControlList   []CIFSShareUserPermissions  `json:"accessControlList,omitempty"`
}

// Create CIFS share info request wrapper
type CreateCIFSShareInfoRequest struct {
  ShareName      string    `json:"shareName,omitempty"`
  AccessControl  CIFSShareUserPermissions `json:"accessControl,omitempty"`
}

// Volume response wrapper
type VolumeResponse struct {
  Name                    string                `json:"name,omitempty"`
  SvmName                 string                `json:"svmName,omitempty"`
  AggregateName           string                `json:"aggregateName,omitempty"`
  Size                    *Capacity             `json:"size,omitempty"`
  UsedSize                *Capacity             `json:"usedSize,omitempty"`
  JunctionPath            string                `json:"junctionPath,omitempty"`
  MountPoint              string                `json:"mountPoint,omitempty"`
  CompressionSpaceSaved   *Capacity             `json:"compressionSpaceSaved,omitempty"`
  DeduplicationSpaceSaved *Capacity             `json:"deduplicationSpaceSaved,omitempty"`
  ThinProvisioning        bool                  `json:"thinProvisioning"`
  Compression             bool                  `json:"compression"`
  Deduplication           bool                  `json:"deduplication"`
  SnapshotPolicy          string                `json:"snapshotPolicy,omitempty"`
  SecurityStyle           string                `json:"securityStyle,omitempty"`
  ExportPolicyInfo        *NamedExportPolicyInfo `json:"exportPolicyInfo,omitempty"`
  ShareNames              []string              `json:"shareNames,omitempty"`
  ShareInfo               []CIFSShareInfo       `json:"shareInfo,omitempty"`
  ParentVolumeName        string                `json:"parentVolumeName,omitempty"`
  RootVolume              bool                  `json:"rootVolume"`
  State                   string                `json:"state,omitempty"`
  VolumeType              string                `json:"volumeType,omitempty"`
  ParentSnapshot          string                `json:"parentSnapshot,omitempty"`
  AutoSizeMode            string                `json:"autoSizeMode,omitempty"`
  MaxGrowSize             *Capacity             `json:"maxGrowSize,omitempty"`
  ProviderVolumeType      string                `json:"providerVolumeType,omitempty"`
  CloneNames              []string              `json:"cloneNames,omitempty"`
  Moving                  bool                  `json:"moving"`
  PrimaryNoFailoverMountPoint   string          `json:"primaryNoFailoverMountPoint,omitempty"`
  SecondaryNoFailoverMountPoint string          `json:"secondaryNoFailoverMountPoint,omitempty"`
  CapacityTier            string                `json:"capacityTier,omitempty"`
}

// Volume clone request wrapper
type VolumeCloneRequest struct {
  NewVolumeName           string  `json:"newVolumeName,omitempty"`
  ParentSnapshot          string  `json:"parentSnapshot,omitempty"`
}

// Volume move request wrapper
type VolumeMoveRequest struct {
  TargetAggregateName     string  `json:"targetAggregateName,omitempty"`
  NumOfDisksToAdd         int     `json:"numOfDisksToAdd,omitempty"`
  CreateTargetAggregate   bool    `json:"createTargetAggregate,omitempty"`
  NewDiskTypeName         string  `json:"newDiskTypeName,omitempty"`
}

// Volume modify request wrapper
type VolumeModifyRequest struct {
  SnapshotPolicyName      string              `json:"snapshotPolicyName,omitempty"`
  ExportPolicyInfo        *ExportPolicyInfo   `json:"exportPolicyInfo,omitempty"`
  ShareInfo               *CIFSShareInfo      `json:"shareInfo,omitempty"`
}

// Volume tier change request wrapper
type ChangeVolumeTierRequest struct {
  AggregateName           string  `json:"aggregateName,omitempty"`
  NumOfDisks              int     `json:"numOfDisks,omitempty"`
  NewAggregate            bool    `json:"newAggregate,omitempty"`
  NewDiskTypeName         string  `json:"newDiskTypeName,omitempty"`
  NewCapacityTier         string  `json:"newCapacityTier,omitempty"`
  IOPS                    int     `json:"iops,omitempty"`
}

func VolumeResponseListFromJSON(data []byte) ([]VolumeResponse, error) {
  var result []VolumeResponse
  err := json.Unmarshal(data, &result)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return result, nil
}
