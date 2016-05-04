package data_types

type SoftLayer_Virtual_GuestInitParameters struct {
	Parameters []interface{} `json:"parameters"`
}

type SoftLayer_Virtual_GuestInit_ImageId_Parameters struct {
	Parameters ImageId_Parameter `json:"parameters"`
}

type ImageId_Parameter struct {
	ImageId int `json:"imageId"`
}
