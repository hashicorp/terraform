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

// The SoftLayer_Dns_Domain data type represents a single DNS domain record hosted on the SoftLayer nameservers. Domains contain general information about the domain name such as name and serial. Individual records such as A, AAAA, CTYPE, and MX records are stored in the domain's associated [[SoftLayer_Dns_Domain_ResourceRecord (type)|SoftLayer_Dns_Domain_ResourceRecord]] records.
type Dns_Domain struct {
	Entity

	// The SoftLayer customer account that owns a domain.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// A domain record's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A flag indicating that the dns domain record is a managed resource.
	ManagedResourceFlag *bool `json:"managedResourceFlag,omitempty" xmlrpc:"managedResourceFlag,omitempty"`

	// A domain's name including top-level domain, for example "example.com".
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of the individual records contained within a domain record. These include but are not limited to A, AAAA, MX, CTYPE, SPF and TXT records.
	ResourceRecordCount *uint `json:"resourceRecordCount,omitempty" xmlrpc:"resourceRecordCount,omitempty"`

	// The individual records contained within a domain record. These include but are not limited to A, AAAA, MX, CTYPE, SPF and TXT records.
	ResourceRecords []Dns_Domain_ResourceRecord `json:"resourceRecords" xmlrpc:"resourceRecords"`

	// The secondary DNS record that defines this domain as being managed through zone transfers.
	Secondary *Dns_Secondary `json:"secondary,omitempty" xmlrpc:"secondary,omitempty"`

	// A unique number denoting the latest revision of a domain. Whenever a domain is changed its corresponding serial number is also changed. Serial numbers typically follow the format yyyymmdd## where yyyy is the current year, mm is the current month, dd is the current day of the month, and ## is the number of the revision for that day. A domain's serial number is automatically updated when edited via the API.
	Serial *int `json:"serial,omitempty" xmlrpc:"serial,omitempty"`

	// The start of authority (SOA) record contains authoritative and propagation details for a DNS zone. This property is not considered in requests to createObject and editObject.
	SoaResourceRecord *Dns_Domain_ResourceRecord_SoaType `json:"soaResourceRecord,omitempty" xmlrpc:"soaResourceRecord,omitempty"`

	// The date that this domain record was last updated.
	UpdateDate *Time `json:"updateDate,omitempty" xmlrpc:"updateDate,omitempty"`
}

// The SoftLayer_Dns_Domain_Forward data type represents a single DNS domain record hosted on the SoftLayer nameservers. Domains contain general information about the domain name such as name and serial. Individual records such as A, AAAA, CTYPE, and MX records are stored in the domain's associated [[SoftLayer_Dns_Domain_ResourceRecord (type)|SoftLayer_Dns_Domain_ResourceRecord]] records.
type Dns_Domain_Forward struct {
	Dns_Domain
}

// The SoftLayer_Dns_Domain_Registration data type represents a domain registration record.
type Dns_Domain_Registration struct {
	Entity

	// The SoftLayer customer account that the domain is registered to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The domain registration status.
	DomainRegistrationStatus *Dns_Domain_Registration_Status `json:"domainRegistrationStatus,omitempty" xmlrpc:"domainRegistrationStatus,omitempty"`

	// no documentation yet
	DomainRegistrationStatusId *int `json:"domainRegistrationStatusId,omitempty" xmlrpc:"domainRegistrationStatusId,omitempty"`

	// The date that the domain registration will expire.
	ExpireDate *Time `json:"expireDate,omitempty" xmlrpc:"expireDate,omitempty"`

	// A domain record's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Indicates whether a domain is locked or unlocked.
	LockedFlag *int `json:"lockedFlag,omitempty" xmlrpc:"lockedFlag,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A domain's name, for example "example.com".
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The registrant verification status.
	RegistrantVerificationStatus *Dns_Domain_Registration_Registrant_Verification_Status `json:"registrantVerificationStatus,omitempty" xmlrpc:"registrantVerificationStatus,omitempty"`

	// no documentation yet
	RegistrantVerificationStatusId *int `json:"registrantVerificationStatusId,omitempty" xmlrpc:"registrantVerificationStatusId,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// no documentation yet
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`
}

// SoftLayer_Dns_Domain_Registration_Registrant_Verification_Status models the state of the registrant. Here are the following status codes:
//
//
// *'''Admin Reviewing''': The registrant data has been submitted and being reviewed by compliance team.
// *'''Pending''': The verification process has been inititated, and verification email will be sent.
// *'''Suspended''': The registrant has failed verification and the domain has been suspended.
// *'''Verified''': The registrant has been validated.
// *'''Verifying''': The verification process has been initiated and is waiting for registrant response.
// *'''Unverified''': The verification process has not been inititated.
//
//
type Dns_Domain_Registration_Registrant_Verification_Status struct {
	Entity

	// The description of the registrant verification status.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique identifier of the registrant verification status
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The unique keyname of the registrant verification status.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of the registrant verification status.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer_Dns_Domain_Registration_Status models the state of domain name. Here are the following status codes:
//
//
// *'''Active''': This domain name is active.
// *'''Pending Owner Approval''': Pending owner approval for completion of transfer.
// *'''Pending Admin Review''': Pending admin review for transfer.
// *'''Pending Registry''': Pending registry for transfer.
// *'''Expired''': Domain name has expired.
//
//
type Dns_Domain_Registration_Status struct {
	Entity

	// The description of the domain registration status names.
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// The unique identifier of the domain registration status
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The unique keyname of the domain registration status.
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// The name of the domain registration status.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Dns_Domain_ResourceRecord data type represents a single resource record entry in a SoftLayer hosted domain. Each resource record contains a ''host'' and ''data'' property, defining a resource's name and it's target data. Domains contain multiple types of resource records. The ''type'' property separates out resource records by type. ''Type'' can take one of the following values:
// * '''"a"''' for [[SoftLayer_Dns_Domain_ResourceRecord_AType|address]] records
// * '''"aaaa"''' for [[SoftLayer_Dns_Domain_ResourceRecord_AaaaType|address]] records
// * '''"cname"''' for [[SoftLayer_Dns_Domain_ResourceRecord_CnameType|canonical name]] records
// * '''"mx"''' for [[SoftLayer_Dns_Domain_ResourceRecord_MxType|mail exchanger]] records
// * '''"ns"''' for [[SoftLayer_Dns_Domain_ResourceRecord_NsType|name server]] records
// * '''"ptr"''' for [[SoftLayer_Dns_Domain_ResourceRecord_PtrType|pointer]] records in reverse domains
// * '''"soa"''' for a domain's [[SoftLayer_Dns_Domain_ResourceRecord_SoaType|start of authority]] record
// * '''"spf"''' for [[SoftLayer_Dns_Domain_ResourceRecord_SpfType|sender policy framework]] records
// * '''"srv"''' for [[SoftLayer_Dns_Domain_ResourceRecord_SrvType|service]] records
// * '''"txt"''' for [[SoftLayer_Dns_Domain_ResourceRecord_TxtType|text]] records
//
//
// As ''SoftLayer_Dns_Domain_ResourceRecord'' objects are created and loaded, the API verifies the ''type'' property and casts the object as the appropriate type.
type Dns_Domain_ResourceRecord struct {
	Entity

	// The value of a domain's resource record. This can be an IP address or a hostname. Fully qualified host and domain name data must end with the "." character.
	Data *string `json:"data,omitempty" xmlrpc:"data,omitempty"`

	// The domain that a resource record belongs to.
	Domain *Dns_Domain `json:"domain,omitempty" xmlrpc:"domain,omitempty"`

	// An identifier belonging to the domain that a resource record is associated with.
	DomainId *int `json:"domainId,omitempty" xmlrpc:"domainId,omitempty"`

	// The amount of time in seconds that a secondary name server (or servers) will hold a zone before it is no longer considered authoritative.
	Expire *int `json:"expire,omitempty" xmlrpc:"expire,omitempty"`

	// The host defined by a resource record. A value of "@" denotes a wildcard.
	Host *string `json:"host,omitempty" xmlrpc:"host,omitempty"`

	// A domain resource record's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Whether the address associated with a PTR record is the gateway address of a subnet.
	IsGatewayAddress *bool `json:"isGatewayAddress,omitempty" xmlrpc:"isGatewayAddress,omitempty"`

	// The amount of time in seconds that a domain's resource records are valid. This is also known as a minimum TTL, and can be overridden by an individual resource record's TTL.
	Minimum *int `json:"minimum,omitempty" xmlrpc:"minimum,omitempty"`

	// Useful in cases where a domain has more than one mail exchanger, the priority property is the priority of the MTA that delivers mail for a domain. A lower number denotes a higher priority, and mail will attempt to deliver through that MTA before moving to lower priority mail servers. Priority is defaulted to 10 upon resource record creation.
	MxPriority *int `json:"mxPriority,omitempty" xmlrpc:"mxPriority,omitempty"`

	// The TCP or UDP port on which the service is to be found.
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// The priority of the target host, lower value means more preferred.
	Priority *int `json:"priority,omitempty" xmlrpc:"priority,omitempty"`

	// The protocol of the desired service; this is usually either TCP or UDP.
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The amount of time in seconds that a secondary name server should wait to check for a new copy of a DNS zone from the domain's primary name server. If a zone file has changed then the secondary DNS server will update it's copy of the zone to match the primary DNS server's zone.
	Refresh *int `json:"refresh,omitempty" xmlrpc:"refresh,omitempty"`

	// The email address of the person responsible for a domain, with the "@" replaced with a ".". For instance, if root@example.org is responsible for example.org, then example.org's SOA responsibility is "root.example.org.".
	ResponsiblePerson *string `json:"responsiblePerson,omitempty" xmlrpc:"responsiblePerson,omitempty"`

	// The amount of time in seconds that a domain's primary name server (or servers) should wait if an attempt to refresh by a secondary name server failed before attempting to refresh a domain's zone with that secondary name server again.
	Retry *int `json:"retry,omitempty" xmlrpc:"retry,omitempty"`

	// The symbolic name of the desired service
	Service *string `json:"service,omitempty" xmlrpc:"service,omitempty"`

	// The Time To Live value of a resource record, measured in seconds. TTL is used by a name server to determine how long to cache a resource record. An SOA record's TTL value defines the domain's overall TTL.
	Ttl *int `json:"ttl,omitempty" xmlrpc:"ttl,omitempty"`

	// A domain resource record's type. A value of "a" denotes an A (address) record, "aaaa" denotes an AAAA (IPv6 address) record, "cname" denotes a CNAME (canonical name) record, "mx" denotes an MX (mail exchanger) record, "ns" denotes an NS (nameserver) record, "ptr" denotes a PTR (pointer/reverse) record, "soa" denotes the SOA (start of authority) record, "spf" denotes a SPF (sender policy framework) record, and "txt" denotes a TXT (text) record. A domain record's type also denotes which class in the SoftLayer API is a best match for extending a resource record.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`

	// A relative weight for records with the same priority.
	Weight *int `json:"weight,omitempty" xmlrpc:"weight,omitempty"`
}

// SoftLayer_Dns_Domain_ResourceRecord_AType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "a" and defines a DNS A record on a SoftLayer hosted domain. An A record directs a host name to an IP address. For instance if the A record for "host.example.org" points to the IP address 10.0.0.1 then the ''host'' property for the A record equals "host" and the ''data'' property equals "10.0.0.1".
type Dns_Domain_ResourceRecord_AType struct {
	Dns_Domain_ResourceRecord
}

// SoftLayer_Dns_Domain_ResourceRecord_AaaaType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "aaaa" and defines a DNS AAAA record on a SoftLayer hosted domain. An AAAA record directs a host name to an IPv6 address. For instance if the AAAA record for "host.example.org" points to the IPv6 address "fe80:0:0:0:0:0:a00:0" then the ''host'' property for the AAAA record equals "host" and the ''data'' property equals "fe80:0:0:0:0:0:a00:0".
type Dns_Domain_ResourceRecord_AaaaType struct {
	Dns_Domain_ResourceRecord
}

// SoftLayer_Dns_Domain_ResourceRecord_CnameType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "cname" and defines a DNS CNAME record on a SoftLayer hosted domain. A CNAME record directs a host name to another host. For instance, if the CNAME record for "alias.example.org" points to the host "host.example.org" then the ''host'' property equals "alias" and the ''data'' property equals "host.example.org.".
//
// DNS entries defined by CNAME should not be used as the data field for an MX record.
type Dns_Domain_ResourceRecord_CnameType struct {
	Dns_Domain_ResourceRecord
}

// SoftLayer_Dns_Domain_ResourceRecord_MxType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "mx" and used to describe MX resource records. MX records control which hosts are responsible as mail exchangers for a domain. For instance, in the domain example.org, an MX record whose host is "@" and data is "mail" says that the host "mail.example.org" is responsible for handling mail for example.org. That means mail sent to users @example.org are delivered to mail.example.org.
//
// Domains can have more than one MX record if it uses more than one server to send mail through. Multiple MX records are denoted by their priority, defined by the mxPriority property.
//
// MX records must be defined for hosts with accompanying A or AAAA resource records. They may not point mail towards a host defined by a CNAME record.
type Dns_Domain_ResourceRecord_MxType struct {
	Dns_Domain_ResourceRecord
}

// SoftLayer_Dns_Domain_ResourceRecord_NsType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "ns" and defines a DNS NS record on a SoftLayer hosted domain. An NS record defines the authoritative name server for a domain. All SoftLayer hosted domains contain NS records for "ns1.softlayer.com" and "ns2.softlayer.com" . For instance, if example.org is hosted on ns1.softlayer.com, then example.org contains an NS record whose ''host'' property equals "@" and whose ''data'' property equals "ns1.example.org".
//
// NS resource records pointing to ns1.softlayer.com or ns2.softlayer.com many not be removed from a SoftLayer hosted domain.
type Dns_Domain_ResourceRecord_NsType struct {
	Dns_Domain_ResourceRecord
}

// SoftLayer_Dns_Domain_ResourceRecord_PtrType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "ptr" and defines a reverse DNS PTR record on the SoftLayer name servers.
//
// The format for a reverse DNS PTR record varies based on whether it is for an IPv4 or IPv6 address.
//
// For an IPv4 address the ''host'' property for every PTR record is the last octet of the IP address that the PTR record belongs to, while the ''data'' property is the canonical name of the host that the reverse lookup resolves to. Every PTR record belongs to a domain on the SoftLayer name servers named by the first three octets of an IP address in reverse order followed by ".in-addr.arpa".
//
// For instance, if the reverse DNS record for 10.0.0.1 is "host.example.org" then it's corresponding SoftLayer_Dns_Domain_ResourceRecord_PtrType host is "1", while it's data property equals "host.example.org". The full name of the reverse record for host.example.org including the domain name is "1.0.0.10.in-addr.arpa".
//
// For an IPv6 address the ''host'' property for every PTR record is the last four octets of the IP address that the PTR record belongs to.  The last four octets need to be in reversed order and each digit separated by a period.  The ''data'' property is the canonical name of the host that the reverse lookup resolves to.  Every PTR record belongs to a domain on the SoftLayer name servers named by the first four octets of an IP address in reverse order, split up by digit with a period, and followed by ".ip6.arpa".
//
// For instance, if the reverse DNS record for fe80:0000:0000:0000:0000:0000:0a00:0001 is "host.example.org" then it's corresponding SoftLayer_Dns_Domain_ResourceRecord_PtrType host is "1.0.0.0.0.0.a.0.0.0.0.0.0.0.0.0", while it's data property equals "host.example.org". The full name of the reverse record for host.example.org including the domain name is "1.0.0.0.0.0.a.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.e.f.ip6.arpa".
//
// PTR record host names may not be changed by [[SoftLayer_Dns_Domain_ResourceRecord::editObject]] or [[SoftLayer_Dns_Domain_ResourceRecord::editObjects]].
type Dns_Domain_ResourceRecord_PtrType struct {
	Dns_Domain_ResourceRecord

	// Whether the address associated with a PTR record is the gateway address of a subnet.
	IsGatewayAddress *bool `json:"isGatewayAddress,omitempty" xmlrpc:"isGatewayAddress,omitempty"`
}

// SoftLayer_Dns_Domain_ResourceRecord_SoaType defines a domains' Start of Authority (or SOA) resource record. A domain's SOA record contains a domain's general and propagation information. Every domain must have one SOA record, and it is not possible to remove a domain's SOA record.
//
// SOA records typically contain a domain's serial number, but the SoftLayer API associates a domain's serial number directly with it's SoftLayer_Dns_Domain record.
type Dns_Domain_ResourceRecord_SoaType struct {
	Dns_Domain_ResourceRecord
}

// SoftLayer_Dns_Domain_ResourceRecord_SpfType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "spf" and defines a DNS SPF record on a SoftLayer hosted domain. An SPF record provides sender policy framework data for a host. For instance, if defining the SPF record "v=spf1 mx:mail.example.org ~all" for "host.example.org". then the ''host'' property equals "host" and the ''data'' property equals "v=spf1 mx:mail.example.org ~all".
//
// SPF records are commonly used in email verification methods such as Sender Policy Framework.
type Dns_Domain_ResourceRecord_SpfType struct {
	Dns_Domain_ResourceRecord_TxtType
}

// SoftLayer_Dns_Domain_ResourceRecord_SrvType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "srv" and defines a DNS SRV record on a SoftLayer hosted domain.
type Dns_Domain_ResourceRecord_SrvType struct {
	Dns_Domain_ResourceRecord

	// The TCP or UDP port on which the service is to be found.
	Port *int `json:"port,omitempty" xmlrpc:"port,omitempty"`

	// The priority of the target host, lower value means more preferred.
	Priority *int `json:"priority,omitempty" xmlrpc:"priority,omitempty"`

	// The protocol of the desired service; this is usually either TCP or UDP.
	Protocol *string `json:"protocol,omitempty" xmlrpc:"protocol,omitempty"`

	// The symbolic name of the desired service
	Service *string `json:"service,omitempty" xmlrpc:"service,omitempty"`

	// A relative weight for records with the same priority.
	Weight *int `json:"weight,omitempty" xmlrpc:"weight,omitempty"`
}

// SoftLayer_Dns_Domain_ResourceRecord_TxtType is a SoftLayer_Dns_Domain_ResourceRecord object whose ''type'' property is set to "txt" and defines a DNS TXT record on a SoftLayer hosted domain. A TXT record provides a text description for a host. For instance, if defining the TXT record "My test host" for "host.example.org". then the ''host'' property equals "host" and the ''data'' property equals "My test host".
//
// TXT records are commonly used in email verification methods such as Sender Policy Framework.
type Dns_Domain_ResourceRecord_TxtType struct {
	Dns_Domain_ResourceRecord
}

// The SoftLayer_Dns_Domain_Reverse data type represents a reverse IP address record.
type Dns_Domain_Reverse struct {
	Dns_Domain

	// Network address the domain is associated with.
	NetworkAddress *string `json:"networkAddress,omitempty" xmlrpc:"networkAddress,omitempty"`
}

// The SoftLayer_Dns_Domain_Reverse_Version4 data type represents a reverse IPv4 address record.
type Dns_Domain_Reverse_Version4 struct {
	Dns_Domain_Reverse
}

// The SoftLayer_Dns_Domain_Reverse_Version6 data type represents a reverse IPv6 address record.
type Dns_Domain_Reverse_Version6 struct {
	Dns_Domain_Reverse
}

// The SoftLayer_Dns_Message data type contains information for a single message generated by the SoftLayer DNS system. SoftLayer_Dns_Messages are typically created during the secondary DNS transfer process.
type Dns_Message struct {
	Entity

	// The date the message was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The domain that is associated with a message.
	Domain *Dns_Domain `json:"domain,omitempty" xmlrpc:"domain,omitempty"`

	// The internal identifier for a DNS message.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The message text.
	Message *string `json:"message,omitempty" xmlrpc:"message,omitempty"`

	// The priority level for a DNS message.  The possible levels are 'notice' and 'error'.
	Priority *string `json:"priority,omitempty" xmlrpc:"priority,omitempty"`

	// The resource record that is associated with a message.
	ResourceRecord *Dns_Domain_ResourceRecord `json:"resourceRecord,omitempty" xmlrpc:"resourceRecord,omitempty"`

	// The secondary DNS record that a message belongs to.
	Secondary *Dns_Secondary `json:"secondary,omitempty" xmlrpc:"secondary,omitempty"`
}

// The SoftLayer_Dns_Secondary data type contains information on a single secondary DNS zone which is managed through SoftLayer's zone transfer service. Domains created via zone transfer may not be modified by the SoftLayer portal or API.
type Dns_Secondary struct {
	Entity

	// The SoftLayer account that owns a secondary DNS record.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The date a secondary DNS record was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The domain record created by zone transfer from a secondary DNS record.
	Domain *Dns_Domain `json:"domain,omitempty" xmlrpc:"domain,omitempty"`

	// A count of the error messages created during secondary DNS record transfer.
	ErrorMessageCount *uint `json:"errorMessageCount,omitempty" xmlrpc:"errorMessageCount,omitempty"`

	// The error messages created during secondary DNS record transfer.
	ErrorMessages []Dns_Message `json:"errorMessages,omitempty" xmlrpc:"errorMessages,omitempty"`

	// The internal identifier for a secondary DNS record.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date when the most recent secondary DNS zone transfer took place.
	LastUpdate *Time `json:"lastUpdate,omitempty" xmlrpc:"lastUpdate,omitempty"`

	// The IP address of the master name server where a secondary DNS zone is transferred from.
	MasterIpAddress *string `json:"masterIpAddress,omitempty" xmlrpc:"masterIpAddress,omitempty"`

	// The current status of the secondary DNS zone.
	Status *Dns_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// The current status of a secondary DNS record.  The status may be one of the following:
	// :*'''0''': Disabled
	// :*'''1''': Active
	// :*'''2''': Transfer Now
	// :*'''3''': An error occurred that prevented the zone transfer from being completed.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// The textual representation of a secondary DNS zone's status.
	StatusText *string `json:"statusText,omitempty" xmlrpc:"statusText,omitempty"`

	// How often a secondary DNS zone should be transferred in minutes.
	TransferFrequency *int `json:"transferFrequency,omitempty" xmlrpc:"transferFrequency,omitempty"`

	// The name of the zone that is transferred.
	ZoneName *string `json:"zoneName,omitempty" xmlrpc:"zoneName,omitempty"`
}

// The SoftLayer_Dns_Status data type contains information for a DNS status
type Dns_Status struct {
	Entity

	// Internal identifier of a DNS status
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// Monitoring DNS status name
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}
