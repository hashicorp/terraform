package compute

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// IPAddressList represents an IP address list.
type IPAddressList struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	IPVersion   string               `json:"ipVersion"`
	State       string               `json:"state"`
	CreateTime  string               `json:"createTime"`
	Addresses   []IPAddressListEntry `json:"ipAddress"`
	ChildLists  []EntitySummary      `json:"childIpAddressList"`
}

// BuildEditRequest creates an EditIPAddressList using the existing addresses and child list references in the IP address list.
func (addressList *IPAddressList) BuildEditRequest() EditIPAddressList {
	edit := &EditIPAddressList{
		Description:  addressList.Description,
		Addresses:    addressList.Addresses,
		ChildListIDs: make([]string, len(addressList.ChildLists)),
	}
	for index, childList := range addressList.ChildLists {
		edit.ChildListIDs[index] = childList.ID
	}

	return *edit
}

// IPAddressListEntry represents an entry in an IP address list.
type IPAddressListEntry struct {
	Begin      string  `json:"begin"`
	End        *string `json:"end,omitempty"`
	PrefixSize *int    `json:"prefixSize,omitempty"`
}

// IPAddressLists represents a page of IPAddressList results.
type IPAddressLists struct {
	AddressLists []IPAddressList `json:"ipAddressList"`

	PagedResult
}

// Request body for creating an IP address list.
type createIPAddressList struct {
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	IPVersion       string               `json:"ipVersion"`
	NetworkDomainID string               `json:"networkDomainId"`
	Addresses       []IPAddressListEntry `json:"ipAddress"`
	ChildListIDs    []string             `json:"childIpAddressListId"`
}

// EditIPAddressList represents the request body for editing an IP address list.
type EditIPAddressList struct {
	ID           string               `json:"id"`
	Description  string               `json:"description"`
	Addresses    []IPAddressListEntry `json:"ipAddress"`
	ChildListIDs []string             `json:"childIpAddressList"`
}

// Request body for deleting an IP address list.
type deleteIPAddressList struct {
	ID string `json:"id"`
}

// GetIPAddressList retrieves the IP address list with the specified Id.
// id is the Id of the IP address list to retrieve.
// Returns nil if no addressList is found with the specified Id.
func (client *Client) GetIPAddressList(id string) (addressList *IPAddressList, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return nil, err
	}

	requestURI := fmt.Sprintf("%s/network/ipAddressList/%s", organizationID, id)
	request, err := client.newRequestV22(requestURI, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		var apiResponse *APIResponseV2

		apiResponse, err = readAPIResponseAsJSON(responseBody, statusCode)
		if err != nil {
			return nil, err
		}

		if apiResponse.ResponseCode == ResponseCodeResourceNotFound {
			return nil, nil // Not an error, but was not found.
		}

		return nil, apiResponse.ToError("Request to retrieve IP address list failed with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	addressList = &IPAddressList{}
	err = json.Unmarshal(responseBody, addressList)

	return addressList, err
}

// ListIPAddressLists retrieves all IP address lists associated with the specified network domain.
func (client *Client) ListIPAddressLists(networkDomainID string) (addressLists *IPAddressLists, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return nil, err
	}

	requestURI := fmt.Sprintf("%s/network/ipAddressList?networkDomainId=%s", organizationID, networkDomainID)
	request, err := client.newRequestV22(requestURI, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		var apiResponse *APIResponseV2

		apiResponse, err = readAPIResponseAsJSON(responseBody, statusCode)
		if err != nil {
			return nil, err
		}

		return nil, apiResponse.ToError("Request to list IP address lists failed with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	addressLists = &IPAddressLists{}
	err = json.Unmarshal(responseBody, addressLists)

	return addressLists, err
}

// CreateIPAddressList creates a new IP address list.
// Returns the Id of the new IP address list.
//
// This operation is synchronous.
func (client *Client) CreateIPAddressList(name string, description string, ipVersion string, networkDomainID string, addresses []IPAddressListEntry, childListIDs []string) (addressListID string, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return "", err
	}

	requestURI := fmt.Sprintf("%s/network/createIpAddressList", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &createIPAddressList{
		Name:            name,
		Description:     description,
		Addresses:       addresses,
		ChildListIDs:    childListIDs,
		NetworkDomainID: networkDomainID,
	})
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return "", err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return "", err
	}

	if apiResponse.ResponseCode != ResponseCodeOK {
		return "", apiResponse.ToError("Request to create IP address list '%s' failed with status code %d (%s): %s", name, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	// Expected: "info" { "name": "ipAddressListId", "value": "the-Id-of-the-new-IP-address-list" }
	if len(apiResponse.FieldMessages) != 1 || apiResponse.FieldMessages[0].FieldName != "ipAddressListId" {
		return "", apiResponse.ToError("Received an unexpected response (missing 'ipAddressListId') with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return apiResponse.FieldMessages[0].Message, nil
}

// EditIPAddressList updates the configuration for a IP address list.
//
// Note that this operation is not additive; it *replaces* the configuration for the IP address list.
// You can IPAddressList.BuildEditRequest() to create an EditIPAddressList request that copies the current state of the IPAddressList (and then apply customisations).
//
// This operation is synchronous.
func (client *Client) EditIPAddressList(id string, edit EditIPAddressList) error {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/network/editIpAddressList", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, edit)
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return err
	}

	if apiResponse.ResponseCode != ResponseCodeOK {
		return apiResponse.ToError("Request to edit IP address list failed with unexpected status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}

// DeleteIPAddressList deletes an existing IP address list.
// Returns an error if the operation was not successful.
//
// This operation is synchronous.
func (client *Client) DeleteIPAddressList(id string) (err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/network/deleteIpAddressList", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &deleteIPAddressList{id})
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return err
	}

	if apiResponse.ResponseCode != ResponseCodeOK {
		return apiResponse.ToError("Request to delete IP address list failed with unexpected status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}
