package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	common "github.com/maximilien/softlayer-go/common"
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
	response, errorCode, err := slvdi.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getObject.json", slvdi.GetName(), vdImageId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Disk_Image{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Disk_Image#getObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Disk_Image{}, errors.New(errorMessage)
	}

	vdImage := datatypes.SoftLayer_Virtual_Disk_Image{}
	err = json.Unmarshal(response, &vdImage)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Disk_Image{}, err
	}

	return vdImage, nil
}
