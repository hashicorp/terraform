package data_types

import "strconv"

type SoftLayer_Product_Item_Price struct {
	Id              int         `json:"id"`
	LocationGroupId int         `json:"locationGroupId"`
	Categories      []Category  `json:"categories,omitempty"`
	Item            *Item       `json:"item,omitempty"`
	Attributes      *Attributes `json:"attributes,omitempty"`
}

type Item struct {
	Id          int    `json:"id"`
	Description string `json:"description"`
	Capacity    string `json:"capacity"`
}

type Category struct {
	Id           int    `json:"id"`
	CategoryCode string `json:"categoryCode"`
}

type Attributes struct {
	Value string `json:"value"`
}

type SoftLayer_Product_Item_Price_Sorted_Data []SoftLayer_Product_Item_Price

func (sorted_data SoftLayer_Product_Item_Price_Sorted_Data) Len() int {
	return len(sorted_data)
}

func (sorted_data SoftLayer_Product_Item_Price_Sorted_Data) Swap(i, j int) {
	sorted_data[i], sorted_data[j] = sorted_data[j], sorted_data[i]
}

func (sorted_data SoftLayer_Product_Item_Price_Sorted_Data) Less(i, j int) bool {
	value1, err := strconv.Atoi(sorted_data[i].Item.Capacity)
	if err != nil {
		return false
	}
	value2, err := strconv.Atoi(sorted_data[j].Item.Capacity)
	if err != nil {
		return false
	}

	return value1 < value2
}
