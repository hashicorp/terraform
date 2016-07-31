package data_types

import (
	"time"
)

type SoftLayer_Virtual_Disk_Image struct {
	Capacity            int        `json:"capacity"`
	Checksum            string     `json:"checksum"`
	CreateDate          *time.Time `json:"createDate"`
	Description         string     `json:"description"`
	Id                  int        `json:"id"`
	ModifyDate          *time.Time `json:"modifyDate"`
	Name                string     `json:"name"`
	ParentId            int        `json:"parentId"`
	StorageRepositoryId int        `json:"storageRepositoryId"`
	TypeId              int        `json:"typeId"`
	Units               string     `json:"units"`
	Uuid                string     `json:"uuid"`
}
