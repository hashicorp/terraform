package data_types

import (
	"time"
)

type SoftLayer_Network_Storage struct {
	AccountId                       int           `json:"accountId,omitempty"`
	CapacityGb                      int           `json:"capacityGb,omitempty"`
	CreateDate                      time.Time     `json:"createDate,omitempty"`
	GuestId                         int           `json:"guestId,omitempty"`
	HardwareId                      int           `json:"hardwareId,omitempty"`
	HostId                          int           `json:"hostId,omitempty"`
	Id                              int           `json:"id,omitempty"`
	NasType                         string        `json:"nasType,omitempty"`
	Notes                           string        `json:"notes,omitempty"`
	Password                        string        `json:"password,omitempty"`
	ServiceProviderId               int           `json:"serviceProviderId,omitempty"`
	UpgradableFlag                  bool          `json:"upgradableFlag,omitempty"`
	Username                        string        `json:"username,omitempty"`
	BillingItem                     *Billing_Item `json:"billingItem,omitempty"`
	LunId                           string        `json:"lunId,omitempty"`
	ServiceResourceBackendIpAddress string        `json:"serviceResourceBackendIpAddress,omitempty"`
}

type Billing_Item struct {
	Id        int         `json:"id,omitempty"`
	OrderItem *Order_Item `json:"orderItem,omitempty"`
}

type Order_Item struct {
	Order *Order `json:"order,omitempty"`
}

type Order struct {
	Id int `json:"id,omitempty"`
}
