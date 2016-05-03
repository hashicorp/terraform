package services

import (
	"bytes"
	"encoding/json"
	"fmt"

	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

type softLayer_Virtual_Disk_Image_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Virtual_Disk_Image_Service(client softlayer.Client) *softLayer_Virtual_Disk_Image_Service {
	return &softLayer_Virtual_Disk_Image_Service{
		client: client,
	}
}

func (slvdi *softLayer_Virtual_Disk_Image_Service) GetName() string {
	return "SoftLayer_Virtual_Disk_Image"
}

func (slvdi *softLayer_Virtual_Disk_Image_Service) GetObject(vdImageId int) (datatypes.SoftLayer_Virtual_Disk_Image, error) {
	response, err := slvdi.client.DoRawHttpRequest(fmt.Sprintf("%s/%d/getObject.json", slvdi.GetName(), vdImageId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Disk_Image{}, err
	}

	vdImage := datatypes.SoftLayer_Virtual_Disk_Image{}
	err = json.Unmarshal(response, &vdImage)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Disk_Image{}, err
	}

	return vdImage, nil
}
