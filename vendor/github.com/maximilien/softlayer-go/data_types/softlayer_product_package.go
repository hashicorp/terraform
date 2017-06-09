package data_types

type Softlayer_Product_Package struct {
	Id          int           `json:"id"`
	Name        string        `json:"name"`
	IsActive    int           `json:"isActive"`
	Description string        `json:"description"`
	PackageType *Package_Type `json:"type"`
}

type Package_Type struct {
	KeyName string `json:"keyName"`
}

type SoftLayer_Product_Item struct {
	Id          int                            `json:"id"`
	Description string                         `json:"description"`
	Capacity    string                         `json:"capacity"`
	Prices      []SoftLayer_Product_Item_Price `json:"prices,omitempty"`
}
