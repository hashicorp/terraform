package data_types

import (
	"time"
)

type SoftLayer_Shh_Key_Parameters struct {
	Parameters []SoftLayer_Security_Ssh_Key `json:"parameters"`
}

type SoftLayer_Security_Ssh_Key struct {
	CreateDate  *time.Time `json:"createDate"`
	Fingerprint string     `json:"fingerprint"`
	Id          int        `json:"id"`
	Key         string     `json:"key"`
	Label       string     `json:"label"`
	ModifyDate  *time.Time `json:"modifyDate"`
	Notes       string     `json:"notes"`
}
