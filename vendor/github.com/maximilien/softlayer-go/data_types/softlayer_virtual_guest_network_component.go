package data_types

import "time"

type SoftLayer_Virtual_Guest_Network_Component struct {
	CreateDate       *time.Time `json:"createDate,omitempty"`
	GuestId          int        `json:"guestId,omitempty"`
	Id               int        `json:"id,omitempty"`
	MacAddress       string     `json:"macAddress,omitempty"`
	MaxSpeed         int        `json:"maxSpeed,omitempty"`
	ModifyDate       *time.Time `json:"modifyDate,omitempty"`
	Name             string     `json:"name,omitempty"`
	NetworkId        int        `json:"networkId,omitempty"`
	Port             int        `json:"port,omitempty"`
	PrimaryIpAddress string     `json:"primaryIpAddress,omitempty"`
	Speed            int        `json:"speed,omitempty"`
	Status           string     `json:"status,omitempty"`
	Uuid             string     `json:"uuid,omitempty"`
}
