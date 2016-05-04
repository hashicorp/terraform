package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	common "github.com/maximilien/softlayer-go/common"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

const (
	NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID = 222
	BLOCK_ITEM_PRICE_ID                    = 40678 // file or block item price id
	CREATE_ISCSI_VOLUME_MAX_RETRY_TIME     = 60
	CREATE_ISCSI_VOLUME_CHECK_INTERVAL     = 5 // seconds
)

type softLayer_Network_Storage_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Network_Storage_Service(client softlayer.Client) *softLayer_Network_Storage_Service {
	return &softLayer_Network_Storage_Service{
		client: client,
	}
}

func (slns *softLayer_Network_Storage_Service) GetName() string {
	return "SoftLayer_Network_Storage"
}

func (slns *softLayer_Network_Storage_Service) CreateIscsiVolume(size int, location string) (datatypes.SoftLayer_Network_Storage, error) {
	if size < 0 {
		return datatypes.SoftLayer_Network_Storage{}, errors.New("Cannot create negative sized volumes")
	}

	sizeItemPriceId, err := slns.getIscsiVolumeItemIdBasedOnSize(size)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	iopsItemPriceId := slns.getPerformanceStorageItemPriceIdByIops(size)

	order := datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi{
		Location:    location,
		ComplexType: "SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi",
		OsFormatType: datatypes.SoftLayer_Network_Storage_Iscsi_OS_Type{
			Id:      12,
			KeyName: "LINUX",
		},
		Prices: []datatypes.SoftLayer_Product_Item_Price{
			datatypes.SoftLayer_Product_Item_Price{
				Id: sizeItemPriceId,
			},
			datatypes.SoftLayer_Product_Item_Price{
				Id: iopsItemPriceId,
			},
			datatypes.SoftLayer_Product_Item_Price{
				Id: BLOCK_ITEM_PRICE_ID,
			},
		},
		PackageId: NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID,
		Quantity:  1,
	}

	productOrderService, err := slns.client.GetSoftLayer_Product_Order_Service()
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	receipt, err := productOrderService.PlaceContainerOrderNetworkPerformanceStorageIscsi(order)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	var iscsiStorage datatypes.SoftLayer_Network_Storage

	for i := 0; i < CREATE_ISCSI_VOLUME_MAX_RETRY_TIME; i++ {
		iscsiStorage, err = slns.findIscsiVolumeId(receipt.OrderId)
		if err == nil {
			break
		} else if i == CREATE_ISCSI_VOLUME_MAX_RETRY_TIME-1 {
			return datatypes.SoftLayer_Network_Storage{}, err
		}

		time.Sleep(CREATE_ISCSI_VOLUME_CHECK_INTERVAL * time.Second)
	}

	return iscsiStorage, nil
}

func (slvgs *softLayer_Network_Storage_Service) DeleteObject(volumeId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d.json", slvgs.GetName(), volumeId), "DELETE", new(bytes.Buffer))

	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to delete volume with id '%d', got '%s' as response from the API.", volumeId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#deleteObject, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slns *softLayer_Network_Storage_Service) DeleteIscsiVolume(volumeId int, immediateCancellationFlag bool) error {

	billingItem, err := slns.GetBillingItem(volumeId)
	if err != nil {
		return err
	}

	if billingItem.Id > 0 {
		billingItemService, err := slns.client.GetSoftLayer_Billing_Item_Service()
		if err != nil {
			return err
		}

		deleted, err := billingItemService.CancelService(billingItem.Id)
		if err != nil {
			return err
		}

		if deleted {
			return nil
		}
	}

	errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage_Service#deleteIscsiVolume with id: '%d'", volumeId)

	return errors.New(errorMessage)
}

func (slns *softLayer_Network_Storage_Service) GetIscsiVolume(volumeId int) (datatypes.SoftLayer_Network_Storage, error) {
	objectMask := []string{
		"accountId",
		"capacityGb",
		"createDate",
		"guestId",
		"hardwareId",
		"hostId",
		"id",
		"nasType",
		"notes",
		"Password",
		"serviceProviderId",
		"upgradableFlag",
		"username",
		"billingItem.id",
		"billingItem.orderItem.order.id",
		"lunId",
		"serviceResourceBackendIpAddress",
	}

	response, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getObject.json", slns.GetName(), volumeId), objectMask, "GET", new(bytes.Buffer))

	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Account#getAccountStatus, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Network_Storage{}, errors.New(errorMessage)
	}

	volume := datatypes.SoftLayer_Network_Storage{}
	err = json.Unmarshal(response, &volume)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	return volume, nil
}

func (slns *softLayer_Network_Storage_Service) GetBillingItem(volumeId int) (datatypes.SoftLayer_Billing_Item, error) {

	response, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getBillingItem.json", slns.GetName(), volumeId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Billing_Item{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_NetWork_Storage#getBillingItem, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Billing_Item{}, errors.New(errorMessage)
	}

	billingItem := datatypes.SoftLayer_Billing_Item{}
	err = json.Unmarshal(response, &billingItem)
	if err != nil {
		return datatypes.SoftLayer_Billing_Item{}, err
	}

	return billingItem, nil
}

func (slns *softLayer_Network_Storage_Service) HasAllowedVirtualGuest(volumeId int, vmId int) (bool, error) {
	filter := string(`{"allowedVirtualGuests":{"id":{"operation":"` + strconv.Itoa(vmId) + `"}}}`)
	response, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequestWithObjectFilterAndObjectMask(fmt.Sprintf("%s/%d/getAllowedVirtualGuests.json", slns.GetName(), volumeId), []string{"id"}, fmt.Sprintf(string(filter)), "GET", new(bytes.Buffer))

	if err != nil {
		return false, errors.New(fmt.Sprintf("Cannot check authentication for volume %d in vm %d", volumeId, vmId))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#hasAllowedVirtualGuest, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	virtualGuest := []datatypes.SoftLayer_Virtual_Guest{}
	err = json.Unmarshal(response, &virtualGuest)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Failed to unmarshal response of checking authentication for volume %d in vm %d", volumeId, vmId))
	}

	if len(virtualGuest) > 0 {
		return true, nil
	}

	return false, nil
}

func (slns *softLayer_Network_Storage_Service) AttachIscsiVolume(virtualGuest datatypes.SoftLayer_Virtual_Guest, volumeId int) (bool, error) {
	parameters := datatypes.SoftLayer_Virtual_Guest_Parameters{
		Parameters: []datatypes.SoftLayer_Virtual_Guest{
			virtualGuest,
		},
	}
	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	resp, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/allowAccessFromVirtualGuest.json", slns.GetName(), volumeId), "PUT", bytes.NewBuffer(requestBody))

	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#attachIscsiVolume, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	allowable, err := strconv.ParseBool(string(resp[:]))
	if err != nil {
		return false, nil
	}

	return allowable, nil
}

func (slns *softLayer_Network_Storage_Service) DetachIscsiVolume(virtualGuest datatypes.SoftLayer_Virtual_Guest, volumeId int) error {
	parameters := datatypes.SoftLayer_Virtual_Guest_Parameters{
		Parameters: []datatypes.SoftLayer_Virtual_Guest{
			virtualGuest,
		},
	}
	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return err
	}

	_, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/removeAccessFromVirtualGuest.json", slns.GetName(), volumeId), "PUT", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Account#getAccountStatus, HTTP error code: '%d'", errorCode)
		return errors.New(errorMessage)
	}

	return nil
}

// Private methods

func (slns *softLayer_Network_Storage_Service) findIscsiVolumeId(orderId int) (datatypes.SoftLayer_Network_Storage, error) {
	ObjectFilter := string(`{"iscsiNetworkStorage":{"billingItem":{"orderItem":{"order":{"id":{"operation":` + strconv.Itoa(orderId) + `}}}}}}`)

	accountService, err := slns.client.GetSoftLayer_Account_Service()
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	iscsiStorages, err := accountService.GetIscsiNetworkStorageWithFilter(ObjectFilter)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	if len(iscsiStorages) == 1 {
		return iscsiStorages[0], nil
	}

	return datatypes.SoftLayer_Network_Storage{}, errors.New(fmt.Sprintf("Cannot find an performance storage (iSCSI volume) with order id %d", orderId))
}

func (slns *softLayer_Network_Storage_Service) getIscsiVolumeItemIdBasedOnSize(size int) (int, error) {
	productPackageService, err := slns.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return 0, err
	}

	itemPrices, err := productPackageService.GetItemPricesBySize(NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID, size)
	if err != nil {
		return 0, err
	}

	var currentItemId int

	if len(itemPrices) > 0 {
		for _, itemPrice := range itemPrices {
			if itemPrice.LocationGroupId == 0 {
				currentItemId = itemPrice.Id
			}
		}
	}

	if currentItemId == 0 {
		return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
	}

	return currentItemId, nil
}

func (slns *softLayer_Network_Storage_Service) getPerformanceStorageItemPriceIdByIops(size int) int {
	switch size {
	case 20:
		return 40838 // 500 IOPS
	case 40:
		return 40988 // 1000 IOPS
	case 80:
		return 41288 // 2000 IOPS
	default:
		return 41788 // 3000 IOPS
	}
}
