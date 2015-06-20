package data_types

type SoftLayer_Virtual_GuestInitParameters struct {
	Parameters SoftLayer_Virtual_GuestInitParameter `json:"parameters"`
}

type SoftLayer_Virtual_GuestInitParameter struct {
	ImageId int `json:"imageId"`
}
