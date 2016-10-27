package data_types

type SoftLayer_Container_Product_Order_Receipt struct {
	OrderId int `json:"orderId"`
}

type SoftLayer_Container_Product_Order_Parameters struct {
	Parameters []SoftLayer_Container_Product_Order `json:"parameters"`
}

type SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi_Parameters struct {
	Parameters []SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi `json:"parameters"`
}

type SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade_Parameters struct {
	Parameters []SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade `json:"parameters"`
}

//http://sldn.softlayer.com/reference/datatypes/SoftLayer_Container_Product_Order
type SoftLayer_Container_Product_Order struct {
	ComplexType   string                         `json:"complexType"`
	Location      string                         `json:"location,omitempty"`
	PackageId     int                            `json:"packageId"`
	Prices        []SoftLayer_Product_Item_Price `json:"prices,omitempty"`
	VirtualGuests []VirtualGuest                 `json:"virtualGuests,omitempty"`
	Properties    []Property                     `json:"properties,omitempty"`
	Quantity      int                            `json:"quantity,omitempty"`
}

//http://sldn.softlayer.com/reference/datatypes/SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi
type SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi struct {
	ComplexType      string                                  `json:"complexType"`
	Location         string                                  `json:"location,omitempty"`
	PackageId        int                                     `json:"packageId"`
	Prices           []SoftLayer_Product_Item_Price          `json:"prices,omitempty"`
	VirtualGuests    []VirtualGuest                          `json:"virtualGuests,omitempty"`
	Properties       []Property                              `json:"properties,omitempty"`
	Quantity         int                                     `json:"quantity,omitempty"`
	OsFormatType     SoftLayer_Network_Storage_Iscsi_OS_Type `json:"osFormatType,omitempty"`
	UseHourlyPricing bool                                    `json:"useHourlyPricing,omitempty"`
}

//http://sldn.softlayer.com/reference/datatypes/SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade
type SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade struct {
	ComplexType   string                         `json:"complexType"`
	Location      string                         `json:"location,omitempty"`
	PackageId     int                            `json:"packageId"`
	Prices        []SoftLayer_Product_Item_Price `json:"prices,omitempty"`
	VirtualGuests []VirtualGuest                 `json:"virtualGuests,omitempty"`
	Properties    []Property                     `json:"properties,omitempty"`
	Quantity      int                            `json:"quantity,omitempty"`
}

type Property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type VirtualGuest struct {
	Id int `json:"id"`
}
