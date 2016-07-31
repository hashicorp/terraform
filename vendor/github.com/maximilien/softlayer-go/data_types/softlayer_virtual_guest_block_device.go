package data_types

import "time"

type SoftLayer_Virtual_Guest_Block_Device struct {
	BootableFlag int        `json:"bootableFlag"`
	CreateDate   *time.Time `json:"createDate"`
	Device       string     `json:"device"`
	DiskImageId  int        `json:"diskImageId"`
	GuestId      int        `json:"guestId"`
	HotPlugFlag  int        `json:"hotPlugFlag"`
	Id           int        `json:"id"`
	ModifyDate   *time.Time `json:"modifyDate"`
	MountMode    string     `json:"mountMode"`
	MountType    string     `json:"mountType"`
	StatusId     int        `json:"statusId"`
	Uuid         string     `json:"uuid"`
}
