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

package datatypes

// no documentation yet
type Security_Certificate struct {
	Entity

	// The number of services currently associated with the certificate.
	AssociatedServiceCount *int `json:"associatedServiceCount,omitempty" xmlrpc:"associatedServiceCount,omitempty"`

	// The certificate provided publicly to clients requesting identity credentials. This certificate is usually signed by a source trusted by the client or a signature chain can be established between this certificate and the truested certificate.
	//
	// This property may only be modified when no services are associated. See associatedServiceCount.
	Certificate *string `json:"certificate,omitempty" xmlrpc:"certificate,omitempty"`

	// The signing request used to request a certificate authority generate a signed certificate.
	//
	// This property may only be modified when no services are associated. See associatedServiceCount.
	CertificateSigningRequest *string `json:"certificateSigningRequest,omitempty" xmlrpc:"certificateSigningRequest,omitempty"`

	// The common name (usually a domain name) encoded within the certificate.
	//
	// This property is read only. Changes made will be silently ignored.
	CommonName *string `json:"commonName,omitempty" xmlrpc:"commonName,omitempty"`

	// The date the certificate _record_ was created. The contents of the certificate may of changed since the record was created, so this does not represent anything about the certificate itself.
	//
	// This property is read only. Changes made will be silently ignored.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The ID of the certificate record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The intermediate certificate authorities certificate that completes the certificate chain for the issued certificate. Required when clients will only trust the root certificate.
	//
	// This property may only be modified when no services are associated. See associatedServiceCount.
	IntermediateCertificate *string `json:"intermediateCertificate,omitempty" xmlrpc:"intermediateCertificate,omitempty"`

	// The size (number of bits) of the public key represented by the certificate.
	KeySize *int `json:"keySize,omitempty" xmlrpc:"keySize,omitempty"`

	// A count of the load balancers virtual IP addresses currently associated with the certificate.
	LoadBalancerVirtualIpAddressCount *uint `json:"loadBalancerVirtualIpAddressCount,omitempty" xmlrpc:"loadBalancerVirtualIpAddressCount,omitempty"`

	// The load balancers virtual IP addresses currently associated with the certificate.
	LoadBalancerVirtualIpAddresses []Network_Application_Delivery_Controller_LoadBalancer_VirtualIpAddress `json:"loadBalancerVirtualIpAddresses,omitempty" xmlrpc:"loadBalancerVirtualIpAddresses,omitempty"`

	// The date the certificate _record_ was last modified.The contents of the certificate may of changed since the record was created, so this does not represent anything about the certificate itself.
	//
	// This property is read only. Changes made will be silently ignored.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A note to help describe the certificate.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// The organizational name encoded in the certificate.
	//
	// This property is read only. Changes made will be silently ignored.
	OrganizationName *string `json:"organizationName,omitempty" xmlrpc:"organizationName,omitempty"`

	// The private key in the key/certificate pair.
	//
	// This property may only be modified when no services are associated. See associatedServiceCount.
	PrivateKey *string `json:"privateKey,omitempty" xmlrpc:"privateKey,omitempty"`

	// The UTC timestamp representing the beginning of the certificate's validity
	//
	// This property is read only. Changes made will be silently ignored.
	ValidityBegin *Time `json:"validityBegin,omitempty" xmlrpc:"validityBegin,omitempty"`

	// The number of days remaining in the validity period for the certificate.
	//
	// This property is read only. Changes made will be silently ignored.
	ValidityDays *int `json:"validityDays,omitempty" xmlrpc:"validityDays,omitempty"`

	// The UTC timestamp representing the end of the certificate's validity period.
	//
	// This property is read only. Changes made will be silently ignored.
	ValidityEnd *Time `json:"validityEnd,omitempty" xmlrpc:"validityEnd,omitempty"`
}

// no documentation yet
type Security_Certificate_Entry struct {
	Entity

	// The ID of the certificate record.
	CertificateId *int `json:"certificateId,omitempty" xmlrpc:"certificateId,omitempty"`

	// The common name (usually a domain name) encoded within the certificate.
	CommonName *string `json:"commonName,omitempty" xmlrpc:"commonName,omitempty"`

	// The size (number of bits) of the public key represented by the certificate.
	KeySize *int `json:"keySize,omitempty" xmlrpc:"keySize,omitempty"`

	// The organizational name encoded in the certificate.
	OrganizationName *string `json:"organizationName,omitempty" xmlrpc:"organizationName,omitempty"`

	// The UTC timestamp representing the beginning of the certificate's validity
	ValidityBegin *Time `json:"validityBegin,omitempty" xmlrpc:"validityBegin,omitempty"`

	// The number of days remaining in the validity period for the certificate.
	ValidityDays *int `json:"validityDays,omitempty" xmlrpc:"validityDays,omitempty"`

	// The UTC timestamp representing the end of the certificate's validity period.
	ValidityEnd *Time `json:"validityEnd,omitempty" xmlrpc:"validityEnd,omitempty"`
}

// SoftLayer_Security_Certificate_Request data type is used to harness your SSL certificate order to a Certificate Authority. This contains data that is required by a Certificate Authority to place an SSL certificate order.
type Security_Certificate_Request struct {
	Entity

	// The account to which a SSL certificate request belongs.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// This is a reference to your SoftLayer account.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// The email address of a person who will approve your SSL certificate order. This is usually an email address of your domain administrator.
	ApproverEmailAddress *string `json:"approverEmailAddress,omitempty" xmlrpc:"approverEmailAddress,omitempty"`

	// The Certificate Authority name
	CertificateAuthorityName *string `json:"certificateAuthorityName,omitempty" xmlrpc:"certificateAuthorityName,omitempty"`

	// A Certificate Signing Request (CSR) string
	CertificateSigningRequest *string `json:"certificateSigningRequest,omitempty" xmlrpc:"certificateSigningRequest,omitempty"`

	// A domain name of a SSL certificate request
	CommonName *string `json:"commonName,omitempty" xmlrpc:"commonName,omitempty"`

	// The date a SSL certificate request was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The date of your SSL certificate went into effect
	EffectiveDate *Time `json:"effectiveDate,omitempty" xmlrpc:"effectiveDate,omitempty"`

	// The expiration date of your SSL certificate
	ExpirationDate *Time `json:"expirationDate,omitempty" xmlrpc:"expirationDate,omitempty"`

	// The internal identifier of an SSL certificate request
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date a SSL certificate request was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The order contains the information related to a SSL certificate request.
	Order *Billing_Order `json:"order,omitempty" xmlrpc:"order,omitempty"`

	// The associated order item for this SSL certificate request.
	OrderItem *Billing_Order_Item `json:"orderItem,omitempty" xmlrpc:"orderItem,omitempty"`

	// The status of a SSL certificate request.
	Status *Security_Certificate_Request_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// A status id reflecting the state of a SSL certificate request
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// The technical contact email address.
	TechnicalContactEmailAddress *string `json:"technicalContactEmailAddress,omitempty" xmlrpc:"technicalContactEmailAddress,omitempty"`
}

// Represents a server type that can be specified when ordering an SSL certificate.
type Security_Certificate_Request_ServerType struct {
	Entity

	// The description of the certificate server type.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The internal identifier of the certificate server type.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The name of the certificate server type.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The value of the certificate server type.
	Value *int `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// Represents the status of an SSL certificate request.
type Security_Certificate_Request_Status struct {
	Entity

	// The description of a SSL certificate request status
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The internal identifier of an SSL certificate request status
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The status name
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer_Security_Directory_Service_Host_Xref_Hardware extends the [[SoftLayer_Security_Directory_Service_Host_Xref]] data type to include hardware specific properties.
type Security_Directory_Service_Host_Xref_Hardware struct {
	Entity

	// The hardware object.
	Host *Hardware `json:"host,omitempty" xmlrpc:"host,omitempty"`
}

// Encryption algorithm intended for use in SSL/TLS communications
type Security_SecureTransportCipher struct {
	Entity

	// Unique identifier for the encryption algorithm
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`
}

// Protocol intended for use in secure communications
type Security_SecureTransportProtocol struct {
	Entity

	// Unique identifier for the protocol
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// List of the supported encryption ciphers
	SupportedSecureTransportCiphers []Security_SecureTransportCipher `json:"supportedSecureTransportCiphers,omitempty" xmlrpc:"supportedSecureTransportCiphers,omitempty"`
}

// no documentation yet
type Security_Ssh_Key struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A count of the image template groups that are linked to an SSH key.
	BlockDeviceTemplateGroupCount *uint `json:"blockDeviceTemplateGroupCount,omitempty" xmlrpc:"blockDeviceTemplateGroupCount,omitempty"`

	// The image template groups that are linked to an SSH key.
	BlockDeviceTemplateGroups []Virtual_Guest_Block_Device_Template_Group `json:"blockDeviceTemplateGroups,omitempty" xmlrpc:"blockDeviceTemplateGroups,omitempty"`

	// The date a ssh key was added.
	//
	// This property is read only. Changes made will be silently ignored.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A short sequence of bytes used to authenticate or lookup a longer ssh key. This will automatically be generated upon adding or modifying the ssh key.
	//
	// This property is read only. Changes made will be silently ignored.
	Fingerprint *string `json:"fingerprint,omitempty" xmlrpc:"fingerprint,omitempty"`

	// The ID of the ssh key record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The ssh key.
	Key *string `json:"key,omitempty" xmlrpc:"key,omitempty"`

	// A descriptive name used to identify a ssh key.
	Label *string `json:"label,omitempty" xmlrpc:"label,omitempty"`

	// The date a ssh key was last modified.
	//
	// This property is read only. Changes made will be silently ignored.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A small note about a ssh key to use at your discretion.
	Notes *string `json:"notes,omitempty" xmlrpc:"notes,omitempty"`

	// A count of the OS root users that are linked to an SSH key.
	SoftwarePasswordCount *uint `json:"softwarePasswordCount,omitempty" xmlrpc:"softwarePasswordCount,omitempty"`

	// The OS root users that are linked to an SSH key.
	SoftwarePasswords []Software_Component_Password `json:"softwarePasswords,omitempty" xmlrpc:"softwarePasswords,omitempty"`
}
