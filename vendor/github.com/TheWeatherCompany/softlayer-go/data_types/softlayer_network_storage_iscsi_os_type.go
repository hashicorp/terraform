package data_types

import (
	"time"
)

type SoftLayer_Network_Storage_Iscsi_OS_Type struct {
	CreateDate time.Time `json:"createDate"`
	Id         int       `json:"id"`
	Name       string    `json:"name"`
	KeyName    string    `json:"keyName"`
}
