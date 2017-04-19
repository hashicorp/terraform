package compute

import (
	"fmt"
	"strconv"
	"strings"
)

const WaitForVolumeReadyTimeout = 600
const WaitForVolumeDeleteTimeout = 600

// StorageVolumeClient is a client for the Storage Volume functions of the Compute API.
type StorageVolumeClient struct {
	ResourceClient
}

// StorageVolumes obtains a StorageVolumeClient which can be used to access to the
// Storage Volume functions of the Compute API
func (c *Client) StorageVolumes() *StorageVolumeClient {
	return &StorageVolumeClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: "storage volume",
			ContainerPath:       "/storage/volume/",
			ResourceRootPath:    "/storage/volume",
		}}

}

type StorageVolumeKind string

const (
	StorageVolumeKindDefault StorageVolumeKind = "/oracle/public/storage/default"
	StorageVolumeKindLatency StorageVolumeKind = "/oracle/public/storage/latency"
)

// StorageVolumeInfo represents information retrieved from the service about a Storage Volume.
type StorageVolumeInfo struct {
	// Shows the default account for your identity domain.
	Account string `json:"account,omitempty"`

	// true indicates that the storage volume can also be used as a boot disk for an instance.
	// If you set the value to true, then you must specify values for the `ImageList` and `ImageListEntry` fields.
	Bootable bool `json:"bootable,omitempty"`

	// The description of the storage volume.
	Description string `json:"description,omitempty"`

	// The hypervisor that this volume is compatible with.
	Hypervisor string `json:"hypervisor,omitempty"`

	// Name of machine image to extract onto this volume when created. This information is provided only for bootable storage volumes.
	ImageList string `json:"imagelist,omitempty"`

	// Specific imagelist entry version to extract.
	ImageListEntry int `json:"imagelist_entry,omitempty"`

	// Three-part name of the machine image. This information is available if the volume is a bootable storage volume.
	MachineImage string `json:"machineimage_name,omitempty"`

	// All volumes are managed volumes. Default value is true.
	Managed bool `json:"managed,omitempty"`

	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`

	// The OS platform this volume is compatible with.
	Platform string `json:"platform,omitempty`

	// The storage-pool property: /oracle/public/storage/latency or /oracle/public/storage/default.
	Properties []string `json:"properties,omitempty"`

	// Boolean field indicating whether this volume can be attached as readonly. If set to False the volume will be attached as read-write.
	ReadOnly bool `json:"readonly,omitempty"`

	// The size of this storage volume in GB.
	Size string `json:"size"`

	// Name of the parent snapshot from which the storage volume is restored or cloned.
	Snapshot string `json:"snapshot,omitempty"`

	// Account of the parent snapshot from which the storage volume is restored.
	SnapshotAccount string `json:"snapshot_account,omitempty"`

	// Id of the parent snapshot from which the storage volume is restored or cloned.
	SnapshotID string `json:"snapshot_id,omitempty"`

	// TODO: this should become a Constant, if/when we have the values
	// The current state of the storage volume.
	Status string `json:"status,omitempty"`

	// Details about the latest state of the storage volume.
	StatusDetail string `json:"status_detail,omitempty"`

	// It indicates the time that the current view of the storage volume was generated.
	StatusTimestamp string `json:"status_timestamp,omitempty"`

	// The storage pool from which this volume is allocated.
	StoragePool string `json:"storage_pool,omitempty"`

	// Comma-separated strings that tag the storage volume.
	Tags []string `json:"tags,omitempty"`

	// Uniform Resource Identifier
	URI string `json:"uri,omitempty"`
}

func (c *StorageVolumeClient) getStorageVolumePath(name string) string {
	return c.getObjectPath("/storage/volume", name)
}

// CreateStorageVolumeInput represents the body of an API request to create a new Storage Volume.
type CreateStorageVolumeInput struct {
	// true indicates that the storage volume can also be used as a boot disk for an instance.
	// If you set the value to true, then you must specify values for the `ImageList` and `ImageListEntry` fields.
	Bootable bool `json:"bootable,omitempty"`

	// The description of the storage volume.
	Description string `json:"description,omitempty"`

	// Name of machine image to extract onto this volume when created. This information is provided only for bootable storage volumes.
	ImageList string `json:"imagelist,omitempty"`

	// Specific imagelist entry version to extract.
	ImageListEntry int `json:"imagelist_entry,omitempty"`

	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`

	// The storage-pool property: /oracle/public/storage/latency or /oracle/public/storage/default.
	Properties []string `json:"properties,omitempty"`

	// The size of this storage volume in GB.
	Size string `json:"size"`

	// Name of the parent snapshot from which the storage volume is restored or cloned.
	Snapshot string `json:"snapshot,omitempty"`

	// Account of the parent snapshot from which the storage volume is restored.
	SnapshotAccount string `json:"snapshot_account,omitempty"`

	// Id of the parent snapshot from which the storage volume is restored or cloned.
	SnapshotID string `json:"snapshot_id,omitempty"`

	// Comma-separated strings that tag the storage volume.
	Tags []string `json:"tags,omitempty"`
}

// CreateStorageVolume uses the given CreateStorageVolumeInput to create a new Storage Volume.
func (c *StorageVolumeClient) CreateStorageVolume(input *CreateStorageVolumeInput) (*StorageVolumeInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.ImageList = c.getQualifiedName(input.ImageList)

	sizeInBytes, err := sizeInBytes(input.Size)
	if err != nil {
		return nil, err
	}
	input.Size = sizeInBytes

	var storageInfo StorageVolumeInfo
	if err := c.createResource(&input, &storageInfo); err != nil {
		return nil, err
	}

	return c.waitForStorageVolumeToBecomeAvailable(input.Name, WaitForVolumeReadyTimeout)
}

// DeleteStorageVolumeInput represents the body of an API request to delete a Storage Volume.
type DeleteStorageVolumeInput struct {
	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`
}

// DeleteStorageVolume deletes the specified storage volume.
func (c *StorageVolumeClient) DeleteStorageVolume(input *DeleteStorageVolumeInput) error {
	if err := c.deleteResource(input.Name); err != nil {
		return err
	}

	return c.waitForStorageVolumeToBeDeleted(input.Name, WaitForVolumeDeleteTimeout)
}

// GetStorageVolumeInput represents the body of an API request to obtain a Storage Volume.
type GetStorageVolumeInput struct {
	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`
}

func (c *StorageVolumeClient) success(result *StorageVolumeInfo) (*StorageVolumeInfo, error) {
	c.unqualify(&result.ImageList)
	c.unqualify(&result.Name)
	c.unqualify(&result.Snapshot)

	sizeInMegaBytes, err := sizeInGigaBytes(result.Size)
	if err != nil {
		return nil, err
	}
	result.Size = sizeInMegaBytes

	return result, nil
}

// GetStorageVolume gets Storage Volume information for the specified storage volume.
func (c *StorageVolumeClient) GetStorageVolume(input *GetStorageVolumeInput) (*StorageVolumeInfo, error) {
	var storageVolume StorageVolumeInfo
	if err := c.getResource(input.Name, &storageVolume); err != nil {
		if WasNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return c.success(&storageVolume)
}

// UpdateStorageVolumeInput represents the body of an API request to update a Storage Volume.
type UpdateStorageVolumeInput struct {
	// The description of the storage volume.
	Description string `json:"description,omitempty"`

	// Name of machine image to extract onto this volume when created. This information is provided only for bootable storage volumes.
	ImageList string `json:"imagelist,omitempty"`

	// Specific imagelist entry version to extract.
	ImageListEntry int `json:"imagelist_entry,omitempty"`

	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`

	// The storage-pool property: /oracle/public/storage/latency or /oracle/public/storage/default.
	Properties []string `json:"properties,omitempty"`

	// The size of this storage volume in GB.
	Size string `json:"size"`

	// Name of the parent snapshot from which the storage volume is restored or cloned.
	Snapshot string `json:"snapshot,omitempty"`

	// Account of the parent snapshot from which the storage volume is restored.
	SnapshotAccount string `json:"snapshot_account,omitempty"`

	// Id of the parent snapshot from which the storage volume is restored or cloned.
	SnapshotID string `json:"snapshot_id,omitempty"`

	// Comma-separated strings that tag the storage volume.
	Tags []string `json:"tags,omitempty"`
}

// UpdateStorageVolume updates the specified storage volume, optionally modifying size, description and tags.
func (c *StorageVolumeClient) UpdateStorageVolume(input *UpdateStorageVolumeInput) (*StorageVolumeInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.ImageList = c.getQualifiedName(input.ImageList)

	sizeInBytes, err := sizeInBytes(input.Size)
	if err != nil {
		return nil, err
	}
	input.Size = sizeInBytes

	path := c.getStorageVolumePath(input.Name)
	_, err = c.executeRequest("PUT", path, input)
	if err != nil {
		return nil, err
	}

	instanceInfo, err := c.waitForStorageVolumeToBecomeAvailable(input.Name, WaitForVolumeReadyTimeout)
	if err != nil {
		return nil, err
	}

	return instanceInfo, nil
}

// waitForStorageVolumeToBecomeAvailable waits until a new Storage Volume is available (i.e. has finished initialising or updating).
func (c *StorageVolumeClient) waitForStorageVolumeToBecomeAvailable(name string, timeoutInSeconds int) (*StorageVolumeInfo, error) {
	var waitResult *StorageVolumeInfo

	err := c.waitFor(
		fmt.Sprintf("storage volume %s to become available", c.getQualifiedName(name)),
		timeoutInSeconds,
		func() (bool, error) {
			getRequest := &GetStorageVolumeInput{
				Name: name,
			}
			result, err := c.GetStorageVolume(getRequest)

			if err != nil {
				return false, err
			}

			if result != nil {
				waitResult = result
				if strings.ToLower(waitResult.Status) == "online" {
					return true, nil
				}
			}

			return false, nil
		})

	return waitResult, err
}

// waitForStorageVolumeToBeDeleted waits until the specified storage volume has been deleted.
func (c *StorageVolumeClient) waitForStorageVolumeToBeDeleted(name string, timeoutInSeconds int) error {
	return c.waitFor(
		fmt.Sprintf("storage volume %s to be deleted", c.getQualifiedName(name)),
		timeoutInSeconds,
		func() (bool, error) {
			getRequest := &GetStorageVolumeInput{
				Name: name,
			}
			result, err := c.GetStorageVolume(getRequest)
			if result == nil {
				return true, nil
			}

			if err != nil {
				return false, err
			}

			return result == nil, nil
		})
}

func sizeInGigaBytes(input string) (string, error) {
	sizeInBytes, err := strconv.Atoi(input)
	if err != nil {
		return "", err
	}
	sizeInKB := sizeInBytes / 1024
	sizeInMB := sizeInKB / 1024
	sizeInGb := sizeInMB / 1024
	return strconv.Itoa(sizeInGb), nil
}

func sizeInBytes(input string) (string, error) {
	sizeInGB, err := strconv.Atoi(input)
	if err != nil {
		return "", err
	}
	sizeInMB := sizeInGB * 1024
	sizeInKB := sizeInMB * 1024
	sizeInBytes := sizeInKB * 1024
	return strconv.Itoa(sizeInBytes), nil
}
