package softlayer

import (
	datatypes "github.com/maximilien/softlayer-go/data_types"
)

type SoftLayer_Virtual_Disk_Image_Service interface {
	Service

	GetObject(id int) (datatypes.SoftLayer_Virtual_Disk_Image, error)
}
