package data_types

import (
	"time"
)

type SoftLayer_Network_Vlan struct {
	AccountId       int        `json:"accountId"`
	Id              int        `json:"Id"`
	ModifyDate      *time.Time `json:"modifyDate,omitempty"`
	Name            string     `json:"name"`
	NetworkVrfId    int        `json:"networkVrfId"`
	Note            string     `json:"note"`
	PrimarySubnetId int        `json:"primarySubnetId"`
	VlanNumber      int        `json:"vlanNumber"`
}
