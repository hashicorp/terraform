package data_types

import (
	"time"
)

type TransactionGroup struct {
	AverageTimeToComplete string `json:"averageTimeToComplete"`
	Name                  string `json:"name"`
}

type TransactionStatus struct {
	AverageDuration string `json:"averageDuration"`
	FriendlyName    string `json:"friendlyName"`
	Name            string `json:"name"`
}

type SoftLayer_Provisioning_Version1_Transaction struct {
	CreateDate       *time.Time `json:"createDate"`
	ElapsedSeconds   int        `json:"elapsedSeconds"`
	GuestId          int        `json:"guestId"`
	HardwareId       int        `json:"hardwareId"`
	Id               int        `json:"id"`
	ModifyDate       *time.Time `json:"modifyDate"`
	StatusChangeDate *time.Time `json:"statusChangeDate"`

	TransactionGroup  TransactionGroup  `json:"transactionGroup,omitempty"`
	TransactionStatus TransactionStatus `json:"transactionStatus,omitempty"`
}
