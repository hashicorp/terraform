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

// The SoftLayer_Ticket data type models a single SoftLayer customer support or notification ticket. Each ticket object contains references to it's updates, the user it's assigned to, the SoftLayer department and employee that it's assigned to, and any hardware objects or attached files associated with the ticket. Tickets are described in further detail on the [[SoftLayer_Ticket]] service page.
//
// To create a support ticket execute the [[SoftLayer_Ticket::createStandardTicket|createStandardTicket]] or [[SoftLayer_Ticket::createAdministrativeTicket|createAdministrativeTicket]] methods in the SoftLayer_Ticket service. To create an upgrade ticket for the SoftLayer sales group execute the [[SoftLayer_Ticket::createUpgradeTicket|createUpgradeTicket]].
type Ticket struct {
	Entity

	// The SoftLayer customer account associated with a ticket.
	Account *Account `json:"account,omitempty" xmlrpc:"account,omitempty"`

	// An internal identifier of the SoftLayer customer account that a ticket is associated with.
	AccountId *int `json:"accountId,omitempty" xmlrpc:"accountId,omitempty"`

	// A count of
	AssignedAgentCount *uint `json:"assignedAgentCount,omitempty" xmlrpc:"assignedAgentCount,omitempty"`

	// no documentation yet
	AssignedAgents []User_Customer `json:"assignedAgents,omitempty" xmlrpc:"assignedAgents,omitempty"`

	// The portal user that a ticket is assigned to.
	AssignedUser *User_Customer `json:"assignedUser,omitempty" xmlrpc:"assignedUser,omitempty"`

	// An internal identifier of the portal user that a ticket is assigned to.
	AssignedUserId *int `json:"assignedUserId,omitempty" xmlrpc:"assignedUserId,omitempty"`

	// A count of the list of additional emails to notify when a ticket update is made.
	AttachedAdditionalEmailCount *uint `json:"attachedAdditionalEmailCount,omitempty" xmlrpc:"attachedAdditionalEmailCount,omitempty"`

	// The list of additional emails to notify when a ticket update is made.
	AttachedAdditionalEmails []User_Customer_AdditionalEmail `json:"attachedAdditionalEmails,omitempty" xmlrpc:"attachedAdditionalEmails,omitempty"`

	// A count of the files attached to a ticket.
	AttachedFileCount *uint `json:"attachedFileCount,omitempty" xmlrpc:"attachedFileCount,omitempty"`

	// The files attached to a ticket.
	AttachedFiles []Ticket_Attachment_File `json:"attachedFiles,omitempty" xmlrpc:"attachedFiles,omitempty"`

	// The hardware associated with a ticket. This is used in cases where a ticket is directly associated with one or more pieces of hardware.
	AttachedHardware []Hardware `json:"attachedHardware,omitempty" xmlrpc:"attachedHardware,omitempty"`

	// no documentation yet
	AttachedHardwareCount *uint `json:"attachedHardwareCount,omitempty" xmlrpc:"attachedHardwareCount,omitempty"`

	// A count of
	AttachedResourceCount *uint `json:"attachedResourceCount,omitempty" xmlrpc:"attachedResourceCount,omitempty"`

	// no documentation yet
	AttachedResources []Ticket_Attachment `json:"attachedResources,omitempty" xmlrpc:"attachedResources,omitempty"`

	// A count of the virtual guests associated with a ticket. This is used in cases where a ticket is directly associated with one or more virtualized guests installations or Virtual Servers.
	AttachedVirtualGuestCount *uint `json:"attachedVirtualGuestCount,omitempty" xmlrpc:"attachedVirtualGuestCount,omitempty"`

	// The virtual guests associated with a ticket. This is used in cases where a ticket is directly associated with one or more virtualized guests installations or Virtual Servers.
	AttachedVirtualGuests []Virtual_Guest `json:"attachedVirtualGuests,omitempty" xmlrpc:"attachedVirtualGuests,omitempty"`

	// Ticket is waiting on a response from a customer flag.
	AwaitingUserResponseFlag *bool `json:"awaitingUserResponseFlag,omitempty" xmlrpc:"awaitingUserResponseFlag,omitempty"`

	// Whether a ticket has a one-time charge associated with it. Standard tickets are free while administrative tickets typically cost $3 USD.
	BillableFlag *bool `json:"billableFlag,omitempty" xmlrpc:"billableFlag,omitempty"`

	// A service cancellation request.
	CancellationRequest *Billing_Item_Cancellation_Request `json:"cancellationRequest,omitempty" xmlrpc:"cancellationRequest,omitempty"`

	// no documentation yet
	ChangeOwnerFlag *bool `json:"changeOwnerFlag,omitempty" xmlrpc:"changeOwnerFlag,omitempty"`

	// The date that a ticket was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A count of
	EmployeeAttachmentCount *uint `json:"employeeAttachmentCount,omitempty" xmlrpc:"employeeAttachmentCount,omitempty"`

	// no documentation yet
	EmployeeAttachments []User_Employee `json:"employeeAttachments,omitempty" xmlrpc:"employeeAttachments,omitempty"`

	// Feedback left by a portal or API user on their experiences in a ticket. Final comments may be created after a ticket is closed.
	FinalComments *string `json:"finalComments,omitempty" xmlrpc:"finalComments,omitempty"`

	// The first physical or virtual server attached to a ticket.
	FirstAttachedResource *Ticket_Attachment `json:"firstAttachedResource,omitempty" xmlrpc:"firstAttachedResource,omitempty"`

	// The first update made to a ticket. This is typically the contents of a ticket when it's created.
	FirstUpdate *Ticket_Update `json:"firstUpdate,omitempty" xmlrpc:"firstUpdate,omitempty"`

	// The SoftLayer department that a ticket is assigned to.
	Group *Ticket_Group `json:"group,omitempty" xmlrpc:"group,omitempty"`

	// The internal identifier of the SoftLayer department that a ticket is assigned to.
	GroupId *int `json:"groupId,omitempty" xmlrpc:"groupId,omitempty"`

	// A ticket's internal identifier. Each ticket is defined by a unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A count of the invoice items associated with a ticket. Ticket based invoice items only exist when a ticket incurs a fee that has been invoiced.
	InvoiceItemCount *uint `json:"invoiceItemCount,omitempty" xmlrpc:"invoiceItemCount,omitempty"`

	// The invoice items associated with a ticket. Ticket based invoice items only exist when a ticket incurs a fee that has been invoiced.
	InvoiceItems []Billing_Invoice_Item `json:"invoiceItems,omitempty" xmlrpc:"invoiceItems,omitempty"`

	// no documentation yet
	LastActivity *Ticket_Activity `json:"lastActivity,omitempty" xmlrpc:"lastActivity,omitempty"`

	// The date that a ticket was last modified. A modification does not necessarily mean that an update was added.
	LastEditDate *Time `json:"lastEditDate,omitempty" xmlrpc:"lastEditDate,omitempty"`

	// The type of user who last edited or updated a ticket. This is either "EMPLOYEE" or "USER".
	LastEditType *string `json:"lastEditType,omitempty" xmlrpc:"lastEditType,omitempty"`

	// no documentation yet
	LastEditor *User_Interface `json:"lastEditor,omitempty" xmlrpc:"lastEditor,omitempty"`

	// The date that the last ticket update was made
	LastResponseDate *Time `json:"lastResponseDate,omitempty" xmlrpc:"lastResponseDate,omitempty"`

	// The last update made to a ticket.
	LastUpdate *Ticket_Update `json:"lastUpdate,omitempty" xmlrpc:"lastUpdate,omitempty"`

	// A timestamp of the last time the Ticket was viewed by the active user.
	LastViewedDate *Time `json:"lastViewedDate,omitempty" xmlrpc:"lastViewedDate,omitempty"`

	// A ticket's associated location within the SoftLayer location hierarchy.
	Location *Location `json:"location,omitempty" xmlrpc:"location,omitempty"`

	// The internal identifier of the location associated with a ticket.
	LocationId *int `json:"locationId,omitempty" xmlrpc:"locationId,omitempty"`

	// The date that a ticket was last updated.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// True if there are new, unread updates to this ticket for the current user, False otherwise.
	NewUpdatesFlag *bool `json:"newUpdatesFlag,omitempty" xmlrpc:"newUpdatesFlag,omitempty"`

	// Whether or not the user who owns a ticket is notified via email when a ticket is updated.
	NotifyUserOnUpdateFlag *bool `json:"notifyUserOnUpdateFlag,omitempty" xmlrpc:"notifyUserOnUpdateFlag,omitempty"`

	// The IP address of the user who opened a ticket.
	OriginatingIpAddress *string `json:"originatingIpAddress,omitempty" xmlrpc:"originatingIpAddress,omitempty"`

	// no documentation yet
	Priority *int `json:"priority,omitempty" xmlrpc:"priority,omitempty"`

	// no documentation yet
	ResponsibleBrandId *int `json:"responsibleBrandId,omitempty" xmlrpc:"responsibleBrandId,omitempty"`

	// A count of
	ScheduledActionCount *uint `json:"scheduledActionCount,omitempty" xmlrpc:"scheduledActionCount,omitempty"`

	// no documentation yet
	ScheduledActions []Provisioning_Version1_Transaction `json:"scheduledActions,omitempty" xmlrpc:"scheduledActions,omitempty"`

	// The amount of money in US Dollars ($USD) that a ticket has charged to an account. A ticket's administrative billing amount is a one time charge and only applies to administrative support tickets.
	ServerAdministrationBillingAmount *int `json:"serverAdministrationBillingAmount,omitempty" xmlrpc:"serverAdministrationBillingAmount,omitempty"`

	// The invoice associated with a ticket. Only tickets with an associated administrative charge have an invoice.
	ServerAdministrationBillingInvoice *Billing_Invoice `json:"serverAdministrationBillingInvoice,omitempty" xmlrpc:"serverAdministrationBillingInvoice,omitempty"`

	// The internal identifier of the invoice associated with a ticket's administrative charge. Only tickets with an administrative charge have an associated invoice.
	ServerAdministrationBillingInvoiceId *int `json:"serverAdministrationBillingInvoiceId,omitempty" xmlrpc:"serverAdministrationBillingInvoiceId,omitempty"`

	// Whether a ticket is a standard or an administrative support ticket. Administrative support tickets typically incur a $3 USD charge.
	ServerAdministrationFlag *int `json:"serverAdministrationFlag,omitempty" xmlrpc:"serverAdministrationFlag,omitempty"`

	// The refund invoice associated with a ticket. Only tickets with a refund applied in them have an associated refund invoice.
	ServerAdministrationRefundInvoice *Billing_Invoice `json:"serverAdministrationRefundInvoice,omitempty" xmlrpc:"serverAdministrationRefundInvoice,omitempty"`

	// The internal identifier of the refund invoice associated with a ticket. Only tickets with an account refund associated with them have an associated refund invoice.
	ServerAdministrationRefundInvoiceId *int `json:"serverAdministrationRefundInvoiceId,omitempty" xmlrpc:"serverAdministrationRefundInvoiceId,omitempty"`

	// no documentation yet
	ServiceProvider *Service_Provider `json:"serviceProvider,omitempty" xmlrpc:"serviceProvider,omitempty"`

	// no documentation yet
	ServiceProviderId *int `json:"serviceProviderId,omitempty" xmlrpc:"serviceProviderId,omitempty"`

	// A ticket's internal identifier at its service provider. Each ticket is defined by a unique identifier.
	ServiceProviderResourceId *int `json:"serviceProviderResourceId,omitempty" xmlrpc:"serviceProviderResourceId,omitempty"`

	// no documentation yet
	State []Ticket_State `json:"state,omitempty" xmlrpc:"state,omitempty"`

	// A count of
	StateCount *uint `json:"stateCount,omitempty" xmlrpc:"stateCount,omitempty"`

	// A ticket's status.
	Status *Ticket_Status `json:"status,omitempty" xmlrpc:"status,omitempty"`

	// A ticket status' internal identifier.
	StatusId *int `json:"statusId,omitempty" xmlrpc:"statusId,omitempty"`

	// A ticket's subject. Only standard support tickets have an associated subject. A standard support ticket's title corresponds with it's subject's name.
	Subject *Ticket_Subject `json:"subject,omitempty" xmlrpc:"subject,omitempty"`

	// An internal identifier of the pre-set subject that a ticket is associated with. Standard support tickets have a subject set while administrative tickets have a null subject. A standard support ticket's title is the name of it's associated subject.
	SubjectId *int `json:"subjectId,omitempty" xmlrpc:"subjectId,omitempty"`

	// A count of
	TagReferenceCount *uint `json:"tagReferenceCount,omitempty" xmlrpc:"tagReferenceCount,omitempty"`

	// no documentation yet
	TagReferences []Tag_Reference `json:"tagReferences,omitempty" xmlrpc:"tagReferences,omitempty"`

	// A ticket's title. This is typically a brief summary of the issue described in the ticket.
	Title *string `json:"title,omitempty" xmlrpc:"title,omitempty"`

	// no documentation yet
	TotalUpdateCount *int `json:"totalUpdateCount,omitempty" xmlrpc:"totalUpdateCount,omitempty"`

	// A count of a ticket's updates.
	UpdateCount *uint `json:"updateCount,omitempty" xmlrpc:"updateCount,omitempty"`

	// A ticket's updates.
	Updates []Ticket_Update `json:"updates,omitempty" xmlrpc:"updates,omitempty"`

	// Whether a user is able to update a ticket.
	UserEditableFlag *bool `json:"userEditableFlag,omitempty" xmlrpc:"userEditableFlag,omitempty"`
}

// no documentation yet
type Ticket_Activity struct {
	Entity

	// no documentation yet
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// no documentation yet
	CreateTimestamp *Time `json:"createTimestamp,omitempty" xmlrpc:"createTimestamp,omitempty"`

	// no documentation yet
	Editor *User_Interface `json:"editor,omitempty" xmlrpc:"editor,omitempty"`

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// no documentation yet
	TicketUpdate *Ticket_Update `json:"ticketUpdate,omitempty" xmlrpc:"ticketUpdate,omitempty"`

	// no documentation yet
	Value *string `json:"value,omitempty" xmlrpc:"value,omitempty"`
}

// SoftLayer tickets have the ability to be associated with specific pieces of hardware in a customer's inventory. Attaching hardware to a ticket can greatly increase response time from SoftLayer for issues that are related to one or more specific servers on a customer's account. The SoftLayer_Ticket_Attachment_Hardware data type models the relationship between a piece of hardware and a ticket. Only one attachment record may exist per hardware item per ticket.
type Ticket_Attachment struct {
	Entity

	// no documentation yet
	AssignedAgent *User_Customer `json:"assignedAgent,omitempty" xmlrpc:"assignedAgent,omitempty"`

	// The internal identifier of an item that is attached to a ticket.
	AttachmentId *int `json:"attachmentId,omitempty" xmlrpc:"attachmentId,omitempty"`

	// The date that an item was attached to a ticket.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// A ticket attachment's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	ScheduledAction *Provisioning_Version1_Transaction `json:"scheduledAction,omitempty" xmlrpc:"scheduledAction,omitempty"`

	// The ticket that an item is attached to.
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// The internal identifier of the ticket that an item is attached to.
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`
}

// no documentation yet
type Ticket_Attachment_Assigned_Agent struct {
	Ticket_Attachment

	// The internal identifier of an assigned Agent that is attached to a ticket.
	AssignedAgentId *int `json:"assignedAgentId,omitempty" xmlrpc:"assignedAgentId,omitempty"`

	// no documentation yet
	Resource *User_Customer `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// This datatype contains tickets referenced from card change request
type Ticket_Attachment_CardChangeRequest struct {
	Ticket_Attachment

	// The card change request that is attached to a ticket.
	Resource *Billing_Payment_Card_ChangeRequest `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// SoftLayer tickets can have have files attached to them. Attaching a file to a ticket is a good way to report issues, provide documentation, and give examples of an issue. Both SoftLayer customers and employees have the ability to attach files to a ticket. The SoftLayer_Ticket_Attachment_File data type models a single file attached to a ticket.
type Ticket_Attachment_File struct {
	Entity

	// The date a file was originally attached to a ticket.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The name of a file attached to a ticket.
	FileName *string `json:"fileName,omitempty" xmlrpc:"fileName,omitempty"`

	// The size of a file attached to a ticket, measured in bytes.
	FileSize *string `json:"fileSize,omitempty" xmlrpc:"fileSize,omitempty"`

	// A ticket file attachment's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The date that a file attachment record was last modified.
	ModifyDate *Time `json:"modifyDate,omitempty" xmlrpc:"modifyDate,omitempty"`

	// no documentation yet
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// The internal identifier of the ticket that a file is attached to.
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`

	// The ticket that a file is attached to.
	Update *Ticket_Update `json:"update,omitempty" xmlrpc:"update,omitempty"`

	// The internal identifier of the ticket update the attached file is associated with.
	UpdateId *int `json:"updateId,omitempty" xmlrpc:"updateId,omitempty"`

	// The internal identifier of the user that uploaded a ticket file attachment. This is only used when A file attachment's ''uploaderType'' is set to "USER".
	UploaderId *string `json:"uploaderId,omitempty" xmlrpc:"uploaderId,omitempty"`

	// The type of user that attached a file to a ticket. This is either "USER" if the file was uploaded by a portal or API user or "EMPLOYEE" if the file was uploaded by a SoftLayer employee.
	UploaderType *string `json:"uploaderType,omitempty" xmlrpc:"uploaderType,omitempty"`
}

// SoftLayer tickets have the ability to be associated with specific pieces of hardware in a customer's inventory. Attaching hardware to a ticket can greatly increase response time from SoftLayer for issues that are related to one or more specific servers on a customer's account. The SoftLayer_Ticket_Attachment_Hardware data type models the relationship between a piece of hardware and a ticket. Only one attachment record may exist per hardware item per ticket.
type Ticket_Attachment_Hardware struct {
	Ticket_Attachment

	// The hardware that is attached to a ticket.
	Hardware *Hardware `json:"hardware,omitempty" xmlrpc:"hardware,omitempty"`

	// The internal identifier of a piece of hardware that is attached to a ticket.
	HardwareId *int `json:"hardwareId,omitempty" xmlrpc:"hardwareId,omitempty"`

	// The hardware that is attached to a ticket.
	Resource *Hardware `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// This datatype contains tickets referenced from manual payments
type Ticket_Attachment_ManualPayment struct {
	Ticket_Attachment

	// The manual payment that is attached to a ticket.
	Resource *Billing_Payment_Card_ManualPayment `json:"resource,omitempty" xmlrpc:"resource,omitempty"`
}

// no documentation yet
type Ticket_Attachment_Scheduled_Action struct {
	Ticket_Attachment

	// no documentation yet
	Resource *Provisioning_Version1_Transaction `json:"resource,omitempty" xmlrpc:"resource,omitempty"`

	// The internal identifier of a scheduled action transaction that is attached to a ticket.
	RunDate *Time `json:"runDate,omitempty" xmlrpc:"runDate,omitempty"`

	// no documentation yet
	Transaction *Provisioning_Version1_Transaction `json:"transaction,omitempty" xmlrpc:"transaction,omitempty"`

	// The internal identifier of a scheduled action transaction that is attached to a ticket.
	TransactionId *int `json:"transactionId,omitempty" xmlrpc:"transactionId,omitempty"`
}

// SoftLayer tickets have the ability to be associated with specific pieces of hardware in a customer's inventory. Attaching hardware to a ticket can greatly increase response time from SoftLayer for issues that are related to one or more specific servers on a customer's account. The SoftLayer_Ticket_Attachment_Hardware data type models the relationship between a piece of hardware and a ticket. Only one attachment record may exist per hardware item per ticket.
type Ticket_Attachment_Virtual_Guest struct {
	Ticket_Attachment

	// The virtualized guest or CloudLayer Computing Instance that is attached to a ticket.
	Resource *Virtual_Guest `json:"resource,omitempty" xmlrpc:"resource,omitempty"`

	// The virtualized guest or CloudLayer Computing Instance that is attached to a ticket.
	VirtualGuest *Virtual_Guest `json:"virtualGuest,omitempty" xmlrpc:"virtualGuest,omitempty"`

	// The internal identifier of the virtualized guest or CloudLayer Computing Instance that is attached to a ticket.
	VirtualGuestId *int `json:"virtualGuestId,omitempty" xmlrpc:"virtualGuestId,omitempty"`
}

// no documentation yet
type Ticket_Chat struct {
	Entity

	// no documentation yet
	Agent *User_Employee `json:"agent,omitempty" xmlrpc:"agent,omitempty"`

	// no documentation yet
	Customer *User_Customer `json:"customer,omitempty" xmlrpc:"customer,omitempty"`

	// no documentation yet
	CustomerId *int `json:"customerId,omitempty" xmlrpc:"customerId,omitempty"`

	// no documentation yet
	EndDate *Time `json:"endDate,omitempty" xmlrpc:"endDate,omitempty"`

	// no documentation yet
	StartDate *Time `json:"startDate,omitempty" xmlrpc:"startDate,omitempty"`

	// no documentation yet
	TicketUpdate *Ticket_Update_Chat `json:"ticketUpdate,omitempty" xmlrpc:"ticketUpdate,omitempty"`

	// no documentation yet
	Transcript *string `json:"transcript,omitempty" xmlrpc:"transcript,omitempty"`
}

// no documentation yet
type Ticket_Chat_Liveperson struct {
	Ticket_Chat
}

// no documentation yet
type Ticket_Chat_TranscriptLine struct {
	Entity

	// no documentation yet
	Speaker *User_Interface `json:"speaker,omitempty" xmlrpc:"speaker,omitempty"`
}

// no documentation yet
type Ticket_Chat_TranscriptLine_Customer struct {
	Ticket_Chat_TranscriptLine
}

// no documentation yet
type Ticket_Chat_TranscriptLine_Employee struct {
	Ticket_Chat_TranscriptLine
}

// SoftLayer tickets have the ability to be assigned to one of SoftLayer's internal departments. The department that a ticket is assigned to is modeled by the SoftLayer_Ticket_Group data type. Ticket groups help to ensure that the proper department is handling a ticket. Standard support tickets are created from a number of pre-determined subjects. These subjects help determine which group a standard ticket is assigned to.
type Ticket_Group struct {
	Entity

	// A count of
	AssignedBrandCount *uint `json:"assignedBrandCount,omitempty" xmlrpc:"assignedBrandCount,omitempty"`

	// no documentation yet
	AssignedBrands []Brand `json:"assignedBrands,omitempty" xmlrpc:"assignedBrands,omitempty"`

	// The category that a ticket group belongs to.
	Category *Ticket_Group_Category `json:"category,omitempty" xmlrpc:"category,omitempty"`

	// A ticket group's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A ticket group's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// The internal identifier for the category that a ticket group belongs to..
	TicketGroupCategoryId *int `json:"ticketGroupCategoryId,omitempty" xmlrpc:"ticketGroupCategoryId,omitempty"`
}

// SoftLayer's support ticket groups represent the department at SoftLayer that is assigned to work one of your support tickets. Many departments are responsible for handling different types of tickets. These types of tickets are modeled in the SoftLayer_Ticket_Group_Category data type. Ticket group categories also help separate differentiate your tickets' issues in the SoftLayer customer portal.
type Ticket_Group_Category struct {
	Entity

	// A ticket group category's unique identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A ticket group category's name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// no documentation yet
type Ticket_Priority struct {
	Entity
}

// no documentation yet
type Ticket_State struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	StateType *Ticket_State_Type `json:"stateType,omitempty" xmlrpc:"stateType,omitempty"`

	// no documentation yet
	StateTypeId *int `json:"stateTypeId,omitempty" xmlrpc:"stateTypeId,omitempty"`

	// no documentation yet
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// no documentation yet
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`
}

// no documentation yet
type Ticket_State_Type struct {
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

// The SoftLayer_Ticket_Status data type models the state of a ticket as it is worked by SoftLayer and its customers. Tickets exist in one of three states:
// *'''OPEN''': Open tickets are considered unresolved issues by SoftLayer and can be assigned to a SoftLayer employee for work. Tickets created by portal or API users are created in the Open state.
// *'''ASSIGNED''': Assigned tickets are identical to open tickets, but are assigned to an individual SoftLayer employee. An assigned ticket is actively being worked by SoftLayer.
// *'''CLOSED''': Tickets are closed when the issue at hand is considered resolved. A SoftLayer employee can change a ticket's status from Closed to Open or Assigned if the need arises.
//
//
// A ticket usually goes from the Open to Assigned to Closed states during its life cycle. If a ticket is forwarded from one department to another it may change from the Assigned state back to Open until it is assigned to a member of the new department.
type Ticket_Status struct {
	Entity

	// A ticket status' internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A ticket status' name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// The SoftLayer_Ticket_Subject data type models one of the possible subjects that a standard support ticket may belong to. A basic support ticket's title matches it's corresponding subject's name.
type Ticket_Subject struct {
	Entity

	// no documentation yet
	Category *Ticket_Subject_Category `json:"category,omitempty" xmlrpc:"category,omitempty"`

	// The subject category id that this ticket subject belongs to.
	CategoryId *int `json:"categoryId,omitempty" xmlrpc:"categoryId,omitempty"`

	// no documentation yet
	Group *Ticket_Group `json:"group,omitempty" xmlrpc:"group,omitempty"`

	// A ticket subject's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A ticket subject's name. This name is used for a standard support ticket's title.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`
}

// SoftLayer_Ticket_Subject_Category groups ticket subjects into logical group.
type Ticket_Subject_Category struct {
	Entity

	// A unique identifier of a ticket subject category.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// A ticket subject category name.
	Name *string `json:"name,omitempty" xmlrpc:"name,omitempty"`

	// A count of
	SubjectCount *uint `json:"subjectCount,omitempty" xmlrpc:"subjectCount,omitempty"`

	// no documentation yet
	Subjects []Ticket_Subject `json:"subjects,omitempty" xmlrpc:"subjects,omitempty"`
}

// no documentation yet
type Ticket_Survey struct {
	Entity
}

// no documentation yet
type Ticket_Type struct {
	Entity

	// no documentation yet
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`
}

// The SoftLayer_Ticket_Update type relates to a single update to a ticket, either by a customer or an employee.
type Ticket_Update struct {
	Entity

	// no documentation yet
	ChangeOwnerActivity *string `json:"changeOwnerActivity,omitempty" xmlrpc:"changeOwnerActivity,omitempty"`

	// The data a ticket update was created.
	CreateDate *Time `json:"createDate,omitempty" xmlrpc:"createDate,omitempty"`

	// The user or SoftLayer employee who created a ticket update.
	Editor *User_Interface `json:"editor,omitempty" xmlrpc:"editor,omitempty"`

	// The internal identifier of the SoftLayer portal or API user who created a ticket update. This is only used if a ticket update's ''editorType'' property is "USER".
	EditorId *int `json:"editorId,omitempty" xmlrpc:"editorId,omitempty"`

	// The type user who created a ticket update. This is either "USER" for an update created by a SoftLayer portal or API user, "EMPLOYEE" for an update created by a SoftLayer employee, or "AUTO" if a ticket update was generated automatically by SoftLayer's backend systems.
	EditorType *string `json:"editorType,omitempty" xmlrpc:"editorType,omitempty"`

	// The contents of a ticket update.
	Entry *string `json:"entry,omitempty" xmlrpc:"entry,omitempty"`

	// The files attached to a ticket update.
	FileAttachment []Ticket_Attachment_File `json:"fileAttachment,omitempty" xmlrpc:"fileAttachment,omitempty"`

	// A count of the files attached to a ticket update.
	FileAttachmentCount *uint `json:"fileAttachmentCount,omitempty" xmlrpc:"fileAttachmentCount,omitempty"`

	// A ticket update's internal identifier.
	Id *int `json:"id,omitempty" xmlrpc:"id,omitempty"`

	// The ticket that a ticket update belongs to.
	Ticket *Ticket `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`

	// The internal identifier of the ticket that a ticket update belongs to.
	TicketId *int `json:"ticketId,omitempty" xmlrpc:"ticketId,omitempty"`

	// The Type of update to this ticket
	Type *Ticket_Update_Type `json:"type,omitempty" xmlrpc:"type,omitempty"`
}

// A SoftLayer_Ticket_Update_Agent type models an update to a ticket made by an agent.
type Ticket_Update_Agent struct {
	Ticket_Update
}

// A SoftLayer_Ticket_Update_Chat is a chat between a customer and a customer service representative relating to a ticket.
type Ticket_Update_Chat struct {
	Ticket_Update

	// The chat between the Customer and Agent
	Chat *Ticket_Chat_Liveperson `json:"chat,omitempty" xmlrpc:"chat,omitempty"`
}

// A SoftLayer_Ticket_Update_Customer is a single update made by a customer to a ticket.
type Ticket_Update_Customer struct {
	Ticket_Update
}

// The SoftLayer_Ticket_Update_Employee data type models an update to a ticket made by a SoftLayer employee.
type Ticket_Update_Employee struct {
	Ticket_Update

	// A ticket update's response rating. Ticket updates posted by SoftLayer employees have the option of earning a rating from SoftLayer's customers. Ratings are based on a 1 - 5 scale, with one being a poor rating while 5 is a very high rating. This is only used if a ticket update's ''editorType'' property is "EMPLOYEE".
	ResponseRating *int `json:"responseRating,omitempty" xmlrpc:"responseRating,omitempty"`
}

// no documentation yet
type Ticket_Update_Type struct {
	Entity

	// no documentation yet
	Description *string `json:"description,omitempty" xmlrpc:"description,omitempty"`

	// no documentation yet
	KeyName *string `json:"keyName,omitempty" xmlrpc:"keyName,omitempty"`

	// no documentation yet
	Ticket *Ticket_Update `json:"ticket,omitempty" xmlrpc:"ticket,omitempty"`
}
