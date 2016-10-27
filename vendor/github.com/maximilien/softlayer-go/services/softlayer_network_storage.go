package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	boshretry "github.com/cloudfoundry/bosh-utils/retrystrategy"
	"github.com/maximilien/softlayer-go/common"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	"github.com/maximilien/softlayer-go/softlayer"
	"github.com/pivotal-golang/clock"
	"os"
)

const (
	NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID = 222
	CREATE_ISCSI_VOLUME_MAX_RETRY_TIME     = 60
	CREATE_ISCSI_VOLUME_CHECK_INTERVAL     = 10 // seconds
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

func (slns *softLayer_Network_Storage_Service) CreateNetworkStorage(size int, capacity int, location string, useHourlyPricing bool) (datatypes.SoftLayer_Network_Storage, error) {
	if size < 0 {
		return datatypes.SoftLayer_Network_Storage{}, errors.New("Cannot create negative sized volumes")
	}

	sizeItemPriceId, err := slns.getIscsiVolumeItemIdBasedOnSize(size)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, err
	}

	var iopsItemPriceId int

	if capacity == 0 {
		iopsItemPriceId, err = slns.selectMediumIopsItemPriceIdOnSize(size)
		if err != nil {
			return datatypes.SoftLayer_Network_Storage{}, err
		}

	} else {
		iopsItemPriceId, err = slns.getItemPriceIdBySizeAndIops(size, capacity)
		if err != nil {
			return datatypes.SoftLayer_Network_Storage{}, err
		}
	}

	blockStorageItemPriceId, err := slns.getBlockStorageItemPriceId()

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
				Id: blockStorageItemPriceId,
			},
		},
		PackageId:        NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID,
		Quantity:         1,
		UseHourlyPricing: useHourlyPricing,
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
	SL_CREATE_ISCSI_VOLUME_TIMEOUT, err := strconv.Atoi(os.Getenv("SL_CREATE_ISCSI_VOLUME_TIMEOUT"))
	if err != nil || SL_CREATE_ISCSI_VOLUME_TIMEOUT == 0 {
		SL_CREATE_ISCSI_VOLUME_TIMEOUT = 600
	}
	SL_CREATE_ISCSI_VOLUME_POLLING_INTERVAL, err := strconv.Atoi(os.Getenv("SL_CREATE_ISCSI_VOLUME_POLLING_INTERVAL"))
	if err != nil || SL_CREATE_ISCSI_VOLUME_POLLING_INTERVAL == 0 {
		SL_CREATE_ISCSI_VOLUME_POLLING_INTERVAL = 10
	}

	execStmtRetryable := boshretry.NewRetryable(
		func() (bool, error) {
			iscsiStorage, err = slns.findIscsiVolumeId(receipt.OrderId)
			if err != nil {
				return true, errors.New(fmt.Sprintf("Failed to find iSCSI volume with id `%d` due to `%s`, retrying...", receipt.OrderId, err.Error()))
			}

			return false, nil
		})
	timeService := clock.NewClock()
	timeoutRetryStrategy := boshretry.NewTimeoutRetryStrategy(time.Duration(SL_CREATE_ISCSI_VOLUME_TIMEOUT)*time.Second, time.Duration(SL_CREATE_ISCSI_VOLUME_POLLING_INTERVAL)*time.Second, execStmtRetryable, timeService, boshlog.NewLogger(boshlog.LevelInfo))
	err = timeoutRetryStrategy.Try()
	if err != nil {
		return datatypes.SoftLayer_Network_Storage{}, errors.New(fmt.Sprintf("Failed to find iSCSI volume with id `%d` after retry within `%d` seconds", receipt.OrderId, SL_CREATE_ISCSI_VOLUME_TIMEOUT))
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

func (slns *softLayer_Network_Storage_Service) DeleteNetworkStorage(volumeId int, immediateCancellationFlag bool) error {

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

func (slns *softLayer_Network_Storage_Service) GetNetworkStorage(volumeId int) (datatypes.SoftLayer_Network_Storage, error) {
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

func (slns *softLayer_Network_Storage_Service) HasAllowedHardware(volumeId int, vmId int) (bool, error) {
	filter := string(`{"allowedVirtualGuests":{"id":{"operation":"` + strconv.Itoa(vmId) + `"}}}`)
	response, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequestWithObjectFilterAndObjectMask(fmt.Sprintf("%s/%d/getAllowedHardware.json", slns.GetName(), volumeId), []string{"id"}, fmt.Sprintf(string(filter)), "GET", new(bytes.Buffer))

	if err != nil {
		return false, errors.New(fmt.Sprintf("Cannot check authentication for volume %d in vm %d", volumeId, vmId))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#hasAllowedHardware, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	hardware := []datatypes.SoftLayer_Hardware{}
	err = json.Unmarshal(response, &hardware)
	if err != nil {
		return false, errors.New(fmt.Sprintf("Failed to unmarshal response of checking authentication for volume %d in vm %d", volumeId, vmId))
	}

	if len(hardware) > 0 {
		return true, nil
	}

	return false, nil
}

func (slns *softLayer_Network_Storage_Service) AttachNetworkStorageToVirtualGuest(virtualGuest datatypes.SoftLayer_Virtual_Guest, volumeId int) (bool, error) {
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
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#attachNetworkStorageToVirtualGuest, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	allowable, err := strconv.ParseBool(string(resp[:]))
	if err != nil {
		return false, nil
	}

	return allowable, nil
}

func (slns *softLayer_Network_Storage_Service) AttachNetworkStorageToHardware(hardware datatypes.SoftLayer_Hardware, volumeId int) (bool, error) {
	parameters := datatypes.SoftLayer_Hardware_Parameters{
		Parameters: []datatypes.SoftLayer_Hardware{
			hardware,
		},
	}
	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	resp, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/allowAccessFromHardware.json", slns.GetName(), volumeId), "PUT", bytes.NewBuffer(requestBody))

	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#attachNetworkStorageToHardware, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	allowable, err := strconv.ParseBool(string(resp[:]))
	if err != nil {
		return false, nil
	}

	return allowable, nil
}

func (slns *softLayer_Network_Storage_Service) DetachNetworkStorageFromVirtualGuest(virtualGuest datatypes.SoftLayer_Virtual_Guest, volumeId int) error {
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
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#detachNetworkStorageToVirtualGuest, HTTP error code: '%d'", errorCode)
		return errors.New(errorMessage)
	}

	return nil
}

func (slns *softLayer_Network_Storage_Service) DetachNetworkStorageFromHardware(hardware datatypes.SoftLayer_Hardware, volumeId int) error {
	parameters := datatypes.SoftLayer_Hardware_Parameters{
		Parameters: []datatypes.SoftLayer_Hardware{
			hardware,
		},
	}
	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return err
	}

	_, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/removeAccessFromHardware.json", slns.GetName(), volumeId), "PUT", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage#detachNetworkStorageToHardware, HTTP error code: '%d'", errorCode)
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

func (slns *softLayer_Network_Storage_Service) getBlockStorageItemPriceId() (int, error) {
	productPackageService, err := slns.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return 0, err
	}

	filters := string(`{"items":{"categories":{"categoryCode":{"operation":"performance_storage_iscsi"}}}}`)
	itemPrices, err := productPackageService.GetItems(NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID, filters)
	if err != nil {
		return 0, err
	}

	if len(itemPrices) > 0 {
		if len(itemPrices[0].Prices) > 0 {
			return itemPrices[0].Prices[0].Id, nil
		}
	}

	return 0, errors.New(fmt.Sprint("No proper block performance storage item price id"))
}

func (slns *softLayer_Network_Storage_Service) getIscsiVolumeItemIdBasedOnSize(size int) (int, error) {
	productPackageService, err := slns.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return 0, err
	}

	keyName := strconv.Itoa(size) + "_GB_PERFORMANCE_STORAGE_SPACE"
	filters := string(`{"itemPrices":{"item":{"keyName":{"operation":"` + keyName + `"}}}}`)
	itemPrices, err := productPackageService.GetItemPrices(NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID, filters)
	if err != nil {
		return 0, err
	}

	var currentItemId int

	if len(itemPrices) > 0 {
		for _, itemPrice := range itemPrices {
			if itemPrice.LocationGroupId == 0 {
				currentItemId = itemPrice.Id
				break
			}
		}
	}

	if currentItemId == 0 {
		return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
	}
	return currentItemId, nil
}

func (slns *softLayer_Network_Storage_Service) getItemPriceIdBySizeAndIops(size int, capacity int) (int, error) {
	productPackageService, err := slns.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return 0, err
	}

	filters := fmt.Sprintf(`{"itemPrices":{"item":{"capacity":{"operation":%d}},"attributes":{"value":{"operation":%d}},"categories":{"categoryCode":{"operation":"performance_storage_iops"}}}}`, capacity, size)
	itemPrices, err := productPackageService.GetItemPrices(NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID, filters)
	if err != nil {
		return 0, err
	}

	var currentItemId int

	if len(itemPrices) > 0 {
		for _, itemPrice := range itemPrices {
			if itemPrice.LocationGroupId == 0 {
				currentItemId = itemPrice.Id
				break
			}
		}
	}

	if currentItemId == 0 {
		return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
	}
	return currentItemId, nil
}

func (slns *softLayer_Network_Storage_Service) selectMaximunIopsItemPriceIdOnSize(size int) (int, error) {
	productPackageService, err := slns.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return 0, err
	}

	filters := fmt.Sprintf(`{"itemPrices":{"attributes":{"value":{"operation":%d}},"categories":{"categoryCode":{"operation":"performance_storage_iops"}}}}`, size)
	itemPrices, err := productPackageService.GetItemPrices(NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID, filters)
	if err != nil {
		return 0, err
	}

	if len(itemPrices) > 0 {
		candidates := filter(itemPrices, func(itemPrice datatypes.SoftLayer_Product_Item_Price) bool {
			return itemPrice.LocationGroupId == 0
		})
		if len(candidates) > 0 {
			sort.Sort(datatypes.SoftLayer_Product_Item_Price_Sorted_Data(candidates))
			return candidates[len(candidates)-1].Id, nil
		} else {
			return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
		}
	}

	return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
}

func (slns *softLayer_Network_Storage_Service) selectMediumIopsItemPriceIdOnSize(size int) (int, error) {
	productPackageService, err := slns.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return 0, err
	}

	filters := fmt.Sprintf(`{"itemPrices":{"attributes":{"value":{"operation":%d}},"categories":{"categoryCode":{"operation":"performance_storage_iops"}}}}`, size)
	itemPrices, err := productPackageService.GetItemPrices(NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID, filters)
	if err != nil {
		return 0, err
	}

	if len(itemPrices) > 0 {
		candidates := filter(itemPrices, func(itemPrice datatypes.SoftLayer_Product_Item_Price) bool {
			return itemPrice.LocationGroupId == 0
		})
		if len(candidates) > 0 {
			sort.Sort(datatypes.SoftLayer_Product_Item_Price_Sorted_Data(candidates))
			return candidates[len(candidates)/2].Id, nil
		} else {
			return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
		}
	}

	return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
}

func (slns *softLayer_Network_Storage_Service) selectMinimumIopsItemPriceIdOnSize(size int) (int, error) {
	productPackageService, err := slns.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return 0, err
	}

	filters := fmt.Sprintf(`{"itemPrices":{"attributes":{"value":{"operation":%d}},"categories":{"categoryCode":{"operation":"performance_storage_iops"}}}}`, size)
	itemPrices, err := productPackageService.GetItemPrices(NETWORK_PERFORMANCE_STORAGE_PACKAGE_ID, filters)
	if err != nil {
		return 0, err
	}

	if len(itemPrices) > 0 {
		candidates := filter(itemPrices, func(itemPrice datatypes.SoftLayer_Product_Item_Price) bool {
			return itemPrice.LocationGroupId == 0
		})
		if len(candidates) > 0 {
			sort.Sort(datatypes.SoftLayer_Product_Item_Price_Sorted_Data(candidates))
			return candidates[0].Id, nil
		} else {
			return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
		}
	}

	return 0, errors.New(fmt.Sprintf("No proper performance storage (iSCSI volume)for size %d", size))
}

func filter(vs []datatypes.SoftLayer_Product_Item_Price, f func(datatypes.SoftLayer_Product_Item_Price) bool) []datatypes.SoftLayer_Product_Item_Price {
	vsf := make([]datatypes.SoftLayer_Product_Item_Price, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}

	return vsf
}
