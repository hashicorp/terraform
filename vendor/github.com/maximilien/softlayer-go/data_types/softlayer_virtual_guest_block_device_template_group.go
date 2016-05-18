package data_types

import "time"

type SoftLayer_Virtual_Guest_Block_Device_Template_Group struct {
	AccountId        int        `json:"accountId"`
	CreateDate       *time.Time `json:"createDate"`
	Id               int        `json:"id"`
	Name             string     `json:"name"`
	Note             string     `json:"note"`
	ParentId         *int       `json:"parentId"`
	PublicFlag       int        `json:"publicFlag"`
	StatusId         int        `json:"statusId"`
	Summary          string     `json:"summary"`
	TransactionId    *int       `json:"transactionId"`
	UserRecordId     int        `json:"userRecordId"`
	GlobalIdentifier string     `json:"globalIdentifier"`
}
