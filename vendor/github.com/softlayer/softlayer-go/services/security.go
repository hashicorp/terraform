/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * AUTOMATICALLY GENERATED CODE - DO NOT MODIFY
 */

package services

import (
	"fmt"
	"strings"

	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/session"
	"github.com/softlayer/softlayer-go/sl"
)

// no documentation yet
type Security_Certificate struct {
	Session *session.Session
	Options sl.Options
}

// GetSecurityCertificateService returns an instance of the Security_Certificate SoftLayer service
func GetSecurityCertificateService(sess *session.Session) Security_Certificate {
	return Security_Certificate{Session: sess}
}

func (r Security_Certificate) Id(id int) Security_Certificate {
	r.Options.Id = &id
	return r
}

func (r Security_Certificate) Mask(mask string) Security_Certificate {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Security_Certificate) Filter(filter string) Security_Certificate {
	r.Options.Filter = filter
	return r
}

func (r Security_Certificate) Limit(limit int) Security_Certificate {
	r.Options.Limit = &limit
	return r
}

func (r Security_Certificate) Offset(offset int) Security_Certificate {
	r.Options.Offset = &offset
	return r
}

// Add a certificate to your account for your records, or for use with various services. Only the certificate and private key are usually required. If your issuer provided an intermediate certificate, you must also provide that certificate. Details will be extracted from the certificate. Validation will be performed between the certificate and the private key as well as the certificate and the intermediate certificate, if provided.
//
// The certificate signing request is not required, but can be provided for your records.
func (r Security_Certificate) CreateObject(templateObject *datatypes.Security_Certificate) (resp datatypes.Security_Certificate, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "createObject", params, &r.Options, &resp)
	return
}

// Remove a certificate from your account. You may not remove a certificate with associated services.
func (r Security_Certificate) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "deleteObject", nil, &r.Options, &resp)
	return
}

// Update a certificate. Modifications are restricted to the note and CSR if the are any services associated with the certificate. There are no modification restrictions for a certificate with no associated services.
func (r Security_Certificate) EditObject(templateObject *datatypes.Security_Certificate) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "editObject", params, &r.Options, &resp)
	return
}

// Locate certificates by their common name, traditionally a domain name.
func (r Security_Certificate) FindByCommonName(commonName *string) (resp []datatypes.Security_Certificate, err error) {
	params := []interface{}{
		commonName,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "findByCommonName", params, &r.Options, &resp)
	return
}

// Retrieve The number of services currently associated with the certificate.
func (r Security_Certificate) GetAssociatedServiceCount() (resp int, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "getAssociatedServiceCount", nil, &r.Options, &resp)
	return
}

// Retrieve The load balancers virtual IP addresses currently associated with the certificate.
func (r Security_Certificate) GetLoadBalancerVirtualIpAddresses() (resp []datatypes.Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "getLoadBalancerVirtualIpAddresses", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Security_Certificate) GetObject() (resp datatypes.Security_Certificate, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve the certificate in PEM (Privacy Enhanced Mail) format, which is a string containing all base64 encoded (DER) certificates delimited by -----BEGIN/END *----- clauses.
func (r Security_Certificate) GetPemFormat() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate", "getPemFormat", nil, &r.Options, &resp)
	return
}

// SoftLayer_Security_Certificate_Request data type is used to harness your SSL certificate order to a Certificate Authority. This contains data that is required by a Certificate Authority to place an SSL certificate order.
type Security_Certificate_Request struct {
	Session *session.Session
	Options sl.Options
}

// GetSecurityCertificateRequestService returns an instance of the Security_Certificate_Request SoftLayer service
func GetSecurityCertificateRequestService(sess *session.Session) Security_Certificate_Request {
	return Security_Certificate_Request{Session: sess}
}

func (r Security_Certificate_Request) Id(id int) Security_Certificate_Request {
	r.Options.Id = &id
	return r
}

func (r Security_Certificate_Request) Mask(mask string) Security_Certificate_Request {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Security_Certificate_Request) Filter(filter string) Security_Certificate_Request {
	r.Options.Filter = filter
	return r
}

func (r Security_Certificate_Request) Limit(limit int) Security_Certificate_Request {
	r.Options.Limit = &limit
	return r
}

func (r Security_Certificate_Request) Offset(offset int) Security_Certificate_Request {
	r.Options.Offset = &offset
	return r
}

// Cancels a pending SSL certificate order at the Certificate Authority
func (r Security_Certificate_Request) CancelSslOrder() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "cancelSslOrder", nil, &r.Options, &resp)
	return
}

// Retrieve The account to which a SSL certificate request belongs.
func (r Security_Certificate_Request) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getAccount", nil, &r.Options, &resp)
	return
}

// Gets the email domains that can be used to validate a certificate to a domain.
func (r Security_Certificate_Request) GetAdministratorEmailDomains(commonName *string) (resp []string, err error) {
	params := []interface{}{
		commonName,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getAdministratorEmailDomains", params, &r.Options, &resp)
	return
}

// Gets the email accounts that can be used to validate a certificate to a domain.
func (r Security_Certificate_Request) GetAdministratorEmailPrefixes() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getAdministratorEmailPrefixes", nil, &r.Options, &resp)
	return
}

// Retrieve The Certificate Authority name
func (r Security_Certificate_Request) GetCertificateAuthorityName() (resp string, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getCertificateAuthorityName", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Security_Certificate_Request) GetObject() (resp datatypes.Security_Certificate_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The order contains the information related to a SSL certificate request.
func (r Security_Certificate_Request) GetOrder() (resp datatypes.Billing_Order, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getOrder", nil, &r.Options, &resp)
	return
}

// Retrieve The associated order item for this SSL certificate request.
func (r Security_Certificate_Request) GetOrderItem() (resp datatypes.Billing_Order_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getOrderItem", nil, &r.Options, &resp)
	return
}

// Returns previous SSL certificate order data. You can use this data for to place a renewal order for a completed SSL certificate.
func (r Security_Certificate_Request) GetPreviousOrderData() (resp datatypes.Container_Product_Order_Security_Certificate, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getPreviousOrderData", nil, &r.Options, &resp)
	return
}

// Returns all the SSL certificate requests.
func (r Security_Certificate_Request) GetSslCertificateRequests(accountId *int) (resp []datatypes.Security_Certificate_Request, err error) {
	params := []interface{}{
		accountId,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getSslCertificateRequests", params, &r.Options, &resp)
	return
}

// Retrieve The status of a SSL certificate request.
func (r Security_Certificate_Request) GetStatus() (resp datatypes.Security_Certificate_Request_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "getStatus", nil, &r.Options, &resp)
	return
}

// A Certificate Authority sends out various emails to your domain administrator or your technical contact. Use this service to have these emails re-sent.
func (r Security_Certificate_Request) ResendEmail(emailType *string) (resp bool, err error) {
	params := []interface{}{
		emailType,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "resendEmail", params, &r.Options, &resp)
	return
}

// Allows you to validate a Certificate Signing Request (CSR) required for an SSL certificate with the certificate authority (CA).  This method sends the CSR, the length of the subscription in months, the certificate type, and the server type for validation against requirements of the CA.  Returns true if valid.
//
// More information on CSR generation can be found at: [http://en.wikipedia.org/wiki/Certificate_signing_request Wikipedia] [https://knowledge.verisign.com/support/ssl-certificates-support/index?page=content&id=AR235&actp=LIST&viewlocale=en_US VeriSign]
func (r Security_Certificate_Request) ValidateCsr(csr *string, validityMonths *int, itemId *int, serverType *string) (resp bool, err error) {
	params := []interface{}{
		csr,
		validityMonths,
		itemId,
		serverType,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request", "validateCsr", params, &r.Options, &resp)
	return
}

// Represents a server type that can be specified when ordering an SSL certificate.
type Security_Certificate_Request_ServerType struct {
	Session *session.Session
	Options sl.Options
}

// GetSecurityCertificateRequestServerTypeService returns an instance of the Security_Certificate_Request_ServerType SoftLayer service
func GetSecurityCertificateRequestServerTypeService(sess *session.Session) Security_Certificate_Request_ServerType {
	return Security_Certificate_Request_ServerType{Session: sess}
}

func (r Security_Certificate_Request_ServerType) Id(id int) Security_Certificate_Request_ServerType {
	r.Options.Id = &id
	return r
}

func (r Security_Certificate_Request_ServerType) Mask(mask string) Security_Certificate_Request_ServerType {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Security_Certificate_Request_ServerType) Filter(filter string) Security_Certificate_Request_ServerType {
	r.Options.Filter = filter
	return r
}

func (r Security_Certificate_Request_ServerType) Limit(limit int) Security_Certificate_Request_ServerType {
	r.Options.Limit = &limit
	return r
}

func (r Security_Certificate_Request_ServerType) Offset(offset int) Security_Certificate_Request_ServerType {
	r.Options.Offset = &offset
	return r
}

// Returns all SSL certificate server types, which are passed in on a [[SoftLayer_Container_Product_Order_Security_Certificate|certificate order]].
func (r Security_Certificate_Request_ServerType) GetAllObjects() (resp []datatypes.Security_Certificate_Request_ServerType, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request_ServerType", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Security_Certificate_Request_ServerType) GetObject() (resp datatypes.Security_Certificate_Request_ServerType, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request_ServerType", "getObject", nil, &r.Options, &resp)
	return
}

// Represents the status of an SSL certificate request.
type Security_Certificate_Request_Status struct {
	Session *session.Session
	Options sl.Options
}

// GetSecurityCertificateRequestStatusService returns an instance of the Security_Certificate_Request_Status SoftLayer service
func GetSecurityCertificateRequestStatusService(sess *session.Session) Security_Certificate_Request_Status {
	return Security_Certificate_Request_Status{Session: sess}
}

func (r Security_Certificate_Request_Status) Id(id int) Security_Certificate_Request_Status {
	r.Options.Id = &id
	return r
}

func (r Security_Certificate_Request_Status) Mask(mask string) Security_Certificate_Request_Status {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Security_Certificate_Request_Status) Filter(filter string) Security_Certificate_Request_Status {
	r.Options.Filter = filter
	return r
}

func (r Security_Certificate_Request_Status) Limit(limit int) Security_Certificate_Request_Status {
	r.Options.Limit = &limit
	return r
}

func (r Security_Certificate_Request_Status) Offset(offset int) Security_Certificate_Request_Status {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Security_Certificate_Request_Status) GetObject() (resp datatypes.Security_Certificate_Request_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request_Status", "getObject", nil, &r.Options, &resp)
	return
}

// Returns all SSL certificate request status objects
func (r Security_Certificate_Request_Status) GetSslRequestStatuses() (resp []datatypes.Security_Certificate_Request_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Certificate_Request_Status", "getSslRequestStatuses", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Security_Ssh_Key struct {
	Session *session.Session
	Options sl.Options
}

// GetSecuritySshKeyService returns an instance of the Security_Ssh_Key SoftLayer service
func GetSecuritySshKeyService(sess *session.Session) Security_Ssh_Key {
	return Security_Ssh_Key{Session: sess}
}

func (r Security_Ssh_Key) Id(id int) Security_Ssh_Key {
	r.Options.Id = &id
	return r
}

func (r Security_Ssh_Key) Mask(mask string) Security_Ssh_Key {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Security_Ssh_Key) Filter(filter string) Security_Ssh_Key {
	r.Options.Filter = filter
	return r
}

func (r Security_Ssh_Key) Limit(limit int) Security_Ssh_Key {
	r.Options.Limit = &limit
	return r
}

func (r Security_Ssh_Key) Offset(offset int) Security_Ssh_Key {
	r.Options.Offset = &offset
	return r
}

// Add a ssh key to your account for use during server provisioning and os reloads.
func (r Security_Ssh_Key) CreateObject(templateObject *datatypes.Security_Ssh_Key) (resp datatypes.Security_Ssh_Key, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Ssh_Key", "createObject", params, &r.Options, &resp)
	return
}

// Remove a ssh key from your account.
func (r Security_Ssh_Key) DeleteObject() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Ssh_Key", "deleteObject", nil, &r.Options, &resp)
	return
}

// Update a ssh key.
func (r Security_Ssh_Key) EditObject(templateObject *datatypes.Security_Ssh_Key) (resp bool, err error) {
	params := []interface{}{
		templateObject,
	}
	err = r.Session.DoRequest("SoftLayer_Security_Ssh_Key", "editObject", params, &r.Options, &resp)
	return
}

// Retrieve
func (r Security_Ssh_Key) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Ssh_Key", "getAccount", nil, &r.Options, &resp)
	return
}

// Retrieve The image template groups that are linked to an SSH key.
func (r Security_Ssh_Key) GetBlockDeviceTemplateGroups() (resp []datatypes.Virtual_Guest_Block_Device_Template_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Ssh_Key", "getBlockDeviceTemplateGroups", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Security_Ssh_Key) GetObject() (resp datatypes.Security_Ssh_Key, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Ssh_Key", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve The OS root users that are linked to an SSH key.
func (r Security_Ssh_Key) GetSoftwarePasswords() (resp []datatypes.Software_Component_Password, err error) {
	err = r.Session.DoRequest("SoftLayer_Security_Ssh_Key", "getSoftwarePasswords", nil, &r.Options, &resp)
	return
}
