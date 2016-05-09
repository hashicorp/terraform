package data_types

import (
	"time"
)

type SoftLayer_Network_Storage_Credential struct {
	AccountId           string    `json:"accountId"`
	CreateDate          time.Time `json:"createDate"`
	Id                  int       `json:"Id"`
	NasCredentialTypeId int       `json:"nasCredentialTypeId"`
	Password            string    `json:"password"`
	Username            string    `json:"username"`
}
