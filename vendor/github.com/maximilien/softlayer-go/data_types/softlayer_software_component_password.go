package data_types

import (
	"time"
)

type Software struct {
	HardwareId                  int    `json:"hardwareId,omitempty"`
	Id                          int    `json:"id"`
	ManufacturerLicenseInstance string `json:"manufacturerLicenseInstance"`
}

type SoftLayer_Software_Component_Password struct {
	CreateDate *time.Time `json:"createDate"`
	Id         int        `json:"id"`
	ModifyDate *time.Time `json:"modifyDate"`
	Notes      string     `json:"notes"`
	Password   string     `json:"password"`
	Port       int        `json:"port"`
	SoftwareId int        `json:"softwareId"`
	Username   string     `json:"username"`

	Software Software `json:"software"`
}
