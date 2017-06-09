package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Product_Package_Service interface {
	Service

	GetItemPrices(packageId int) ([]datatypes.SoftLayer_Product_Item_Price, error)
	GetItemPricesBySize(packageId int, size int) ([]datatypes.SoftLayer_Product_Item_Price, error)
	GetItems(packageId int) ([]datatypes.SoftLayer_Product_Item, error)
	GetItemsByType(packageType string) ([]datatypes.SoftLayer_Product_Item, error)

	GetPackagesByType(packageType string) ([]datatypes.Softlayer_Product_Package, error)
	GetOnePackageByType(packageType string) (datatypes.Softlayer_Product_Package, error)
}
