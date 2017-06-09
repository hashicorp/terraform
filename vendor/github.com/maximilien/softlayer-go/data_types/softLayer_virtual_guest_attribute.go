package data_types

type SoftLayer_Virtual_Guest_Attribute_Type struct {
	Keyname string `json:"keyname"`
	Name    string `json:"name"`
}

type SoftLayer_Virtual_Guest_Attribute struct {
	Value string `json:"value"`

	Type SoftLayer_Virtual_Guest_Attribute_Type `json:"type"`
}
