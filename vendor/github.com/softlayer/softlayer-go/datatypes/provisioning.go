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

// The SoftLayer_Provisioning_Hook contains all the information needed to add a hook into a server/Virtual provision and os reload.
type Provisioning_Hook struct {
	Entity

	// no documentation yet
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The ID of the account the script belongs to.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	HookType *Provisioning_Hook_Type `json:"hookType,omitempty" xmlrpc:"hookType,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// The name of the hook.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The ID of the type of hook the script is identified as.  Currently only CUSTOMER_PROVIDED_HOOK has been implemented.
	TypeId *int `json:"typeId,omitempty" xmlrpc:"typeId,omitempty"`

	// The endpoint that the script will be downloaded from (USERNAME AND PASSWORD SHOULD BE INCLUDED HERE).  If the endpoint is HTTP, the script will only be downloaded.  If the endpoint is HTTPS, the script will be downloaded and executed.
	Uri *string `json:"uri,omitempty" xmlrpc:"uri,omitempty"`
}

// no documentation yet
type Provisioning_Hook_Type struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Provisioning_Maintenance_Classification represent a maintenance type for the specific hardware maintenance desired.
type Provisioning_Maintenance_Classification struct {
	Entity

	// The id of the maintenance classification.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ItemCategories []Provisioning_Maintenance_Classification_Item_Category `json:"itemCategories,omitempty" xmlrpc:"itemCategories,omitempty"`

	// A count of
	ItemCategoryCount *uint `json:"itemCategoryCount,omitempty" xmlrpc:"itemCategoryCount,omitempty"`

	// The number of slots required for the maintenance classification.
	Slots *int `json:"slots,omitempty" xmlrpc:"slots,omitempty"`

	// The type or name of the maintenance classification.
	Type *string `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// no documentation yet
type Provisioning_Maintenance_Classification_Item_Category struct {
	Entity

	// no documentation yet
	ItemCategoryId *int `json:"itemCategoryId,omitempty" xmlrpc:"itemCategoryId,omitempty"`

	// no documentation yet
	MaintenanceClassification *Provisioning_Maintenance_Classification `json:"maintenanceClassification,omitempty" xmlrpc:"maintenanceClassification,omitempty"`

	// no documentation yet
	MaintenanceClassificationId *int `json:"maintenanceClassificationId,omitempty" xmlrpc:"maintenanceClassificationId,omitempty"`
}

// The SoftLayer_Provisioning_Maintenance_Slots represent the available slots for a given maintenance window at a SoftLayer data center.
type Provisioning_Maintenance_Slots struct {
	Entity

	// The available slots for a maintenance window.
	AvailableSlots *int `json:"availableSlots,omitempty" xmlrpc:"availableSlots,omitempty"`
}

// no documentation yet
type Provisioning_Maintenance_Ticket struct {
	Entity

	// no documentation yet
	AvailableSlots *Provisioning_Maintenance_Slots `json:"availableSlots,omitempty" xmlrpc:"availableSlots,omitempty"`

	// no documentation yet
	MaintClassId *int `json:"maintClassId,omitempty" xmlrpc:"maintClassId,omitempty"`

	// no documentation yet
	MaintWindowId *int `json:"maintWindowId,omitempty" xmlrpc:"maintWindowId,omitempty"`

	// no documentation yet
	MaintenanceClass *Provisioning_Maintenance_Classification `json:"maintenanceClass,omitempty" xmlrpc:"maintenanceClass,omitempty"`

	// no documentation yet
	MaintenanceDate *Time `json:"maintenanceDate,omitempty" xmlrpc:"maintenanceDate,omitempty"`

	// no documentation yet
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// no documentation yet
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`
}

// The SoftLayer_Provisioning_Maintenance_Window represent a time window that SoftLayer performs a hardware or software maintenance and upgrades.
type Provisioning_Maintenance_Window struct {
	Entity

	// The date and time a maintenance window is scheduled to begin.
	BeginDate *Time `json:"beginDate,omitempty" xmlrpc:"beginDate,omitempty"`

	// An ISO-8601 numeric representation of the day of the week that a maintenance window is performed. 1: Monday, 7: Sunday
	DayOfWeek *int `json:"dayOfWeek,omitempty" xmlrpc:"dayOfWeek,omitempty"`

	// The date and time a maintenance window is scheduled to end.
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// Id of the maintenance window
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// An internal identifier of the location (data center) record that a maintenance window takes place in.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// An internal identifier of the datacenter timezone.
	PortalTzId *int `json:"portalTzId,omitempty" xmlrpc:"portalTzId,omitempty"`
}

// The SoftLayer_Provisioning_Version1_Transaction data type contains general information relating to a single SoftLayer hardware transaction.
//
// SoftLayer customers are unable to change their hardware transactions.
type Provisioning_Version1_Transaction struct {
	Entity

	// The account that a transaction belongs to.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// The date a transaction was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The amount of seconds that have elapsed since the transaction was last modified.
	ElapsedSeconds *int `json:"elapsedSeconds,omitempty" xmlrpc:"elapsedSeconds,omitempty"`

	// The guest record for this transaction.
	Guest *Virtual_Guest `json:"guest,omitempty" xmlrpc:"guest,omitempty"`

	// A transaction's associated guest identification number.
	GuestId *int `json:"guestId,omitempty" xmlrpc:"guestId,omitempty"`

	// The hardware object for this transaction.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// A transaction's associated hardware identification number.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// A transaction's identifying number.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Loopback []Provisioning_Version1_Transaction `json:"loopback,omitempty" xmlrpc:"loopback,omitempty"`

	// A count of
	LoopbackCount *uint `json:"loopbackCount,omitempty" xmlrpc:"loopbackCount,omitempty"`

	// The date a transaction was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// A count of
	PendingTransactionCount *uint `json:"pendingTransactionCount,omitempty" xmlrpc:"pendingTransactionCount,omitempty"`

	// no documentation yet
	PendingTransactions []Provisioning_Version1_Transaction `json:"pendingTransactions,omitempty" xmlrpc:"pendingTransactions,omitempty"`

	// The date the transaction status was last modified.
	StatusChangeDate *Time `json:"statusChangeDate,omitempty" xmlrpc:"statusChangeDate,omitempty"`

	// no documentation yet
	TicketScheduledActionReference []Ticket_Attachment `json:"ticketScheduledActionReference,omitempty" xmlrpc:"ticketScheduledActionReference,omitempty"`

	// A count of
	TicketScheduledActionReferenceCount *uint `json:"ticketScheduledActionReferenceCount,omitempty" xmlrpc:"ticketScheduledActionReferenceCount,omitempty"`

	// A transaction's group. This group object determines what type of service is being done on the hardware.
	TransactionGroup *Provisioning_Version1_Transaction_Group `json:"transactionGroup,omitempty" xmlrpc:"transactionGroup,omitempty"`

	// A transaction's status. This status object determines the state it is in the transaction group.
	TransactionStatus *Provisioning_Version1_Transaction_Status `json:"transactionStatus,omitempty" xmlrpc:"transactionStatus,omitempty"`
}

// The SoftLayer_Provisioning_Version1_Transaction_Group data type contains general information relating to a single SoftLayer hardware transaction group.
//
// SoftLayer customers are unable to change their hardware transactions or the hardware transaction group.
type Provisioning_Version1_Transaction_Group struct {
	Entity

	// Average time, in minutes, for this type of transaction to complete. Please note that this is only an estimate.
	AverageTimeToComplete *Float64 `json:"averageTimeToComplete,omitempty" xmlrpc:"averageTimeToComplete,omitempty"`

	// A transaction group's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Provisioning_Version1_Transaction_History struct {
	Entity

	// The finish date of a transaction history record.
	FinishDate *Time `json:"finishDate,omitempty" xmlrpc:"finishDate,omitempty"`

	// The guest from where transaction history originates.
	Guest *Virtual_Guest `json:"guest,omitempty" xmlrpc:"guest,omitempty"`

	// The guest ID associated with a transaction history.
	GuestId *int `json:"guestId,omitempty" xmlrpc:"guestId,omitempty"`

	// The hardware from where transaction history originates.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The hardware ID associated with a transaction history.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// The host ID associated with a transaction history.
	HostId *int `json:"hostId,omitempty" xmlrpc:"hostId,omitempty"`

	// The ID associated with a transaction history.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The start date of a transaction history record.
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// The transaction from where transaction history originates.
	Transaction *Provisioning_Version1_Transaction `json:"transaction,omitempty" xmlrpc:"transaction,omitempty"`

	// The transaction ID associated with a transaction history.
	TransactionId *int `json:"transactionId,omitempty" xmlrpc:"transactionId,omitempty"`

	// The transaction status of a transaction history.
	TransactionStatus *Provisioning_Version1_Transaction_Status `json:"transactionStatus,omitempty" xmlrpc:"transactionStatus,omitempty"`

	// The transaction status ID associated with a transaction history.
	TransactionStatusId *int `json:"transactionStatusId,omitempty" xmlrpc:"transactionStatusId,omitempty"`
}

// The SoftLayer_Provisioning_Version1_Transaction_Status data type contains general information relating to a single SoftLayer hardware transaction status.
//
// SoftLayer customers are unable to change their hardware transaction status.
type Provisioning_Version1_Transaction_Status struct {
	Entity

	// Hardware transaction status average duration.
	AverageDuration *Float64 `json:"averageDuration,omitempty" xmlrpc:"averageDuration,omitempty"`

	// Transaction status friendly name.
	FriendlyName *string `json:"friendlyName,omitempty" xmlrpc:"friendlyName,omitempty"`

	// Transaction status name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of
	NonCompletedTransactionCount *uint `json:"nonCompletedTransactionCount,omitempty" xmlrpc:"nonCompletedTransactionCount,omitempty"`

	// no documentation yet
	NonCompletedTransactions []Provisioning_Version1_Transaction `json:"nonCompletedTransactions,omitempty" xmlrpc:"nonCompletedTransactions,omitempty"`
}

// no documentation yet
type Provisioning_Version1_Transaction_SubnetMigration struct {
	Provisioning_Version1_Transaction
}
