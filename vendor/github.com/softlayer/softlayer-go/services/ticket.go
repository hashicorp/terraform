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

// The SoftLayer_Ticket data type models a single SoftLayer customer support or notification ticket. Each ticket object contains references to it's updates, the user it's assigned to, the SoftLayer department and employee that it's assigned to, and any hardware objects or attached files associated with the ticket. Tickets are described in further detail on the [[SoftLayer_Ticket]] service page.
//
// To create a support ticket execute the [[SoftLayer_Ticket::createStandardTicket|createStandardTicket]] or [[SoftLayer_Ticket::createAdministrativeTicket|createAdministrativeTicket]] methods in the SoftLayer_Ticket service. To create an upgrade ticket for the SoftLayer sales group execute the [[SoftLayer_Ticket::createUpgradeTicket|createUpgradeTicket]].
type Ticket struct {
	Session *session.Session
	Options sl.Options
}

// GetTicketService returns an instance of the Ticket SoftLayer service
func GetTicketService(sess *session.Session) Ticket {
	return Ticket{Session: sess}
}

func (r Ticket) Id(id int) Ticket {
	r.Options.Id = &id
	return r
}

func (r Ticket) Mask(mask string) Ticket {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Ticket) Filter(filter string) Ticket {
	r.Options.Filter = filter
	return r
}

func (r Ticket) Limit(limit int) Ticket {
	r.Options.Limit = &limit
	return r
}

func (r Ticket) Offset(offset int) Ticket {
	r.Options.Offset = &offset
	return r
}

//
//
//
func (r Ticket) AddAssignedAgent(agentId *int) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		agentId,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addAssignedAgent", params, &r.Options, &resp)
	return
}

// Creates new additional emails for assigned user if new emails are provided. Attaches any newly created additional emails to ticket.
func (r Ticket) AddAttachedAdditionalEmails(emails []string) (resp bool, err error) {
	params := []interface{}{
		emails,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addAttachedAdditionalEmails", params, &r.Options, &resp)
	return
}

// Attach the given file to a SoftLayer ticket. A file attachment is a convenient way to submit non-textual error reports to SoftLayer employees in a ticket. File attachments to tickets must have a unique name.
func (r Ticket) AddAttachedFile(fileAttachment *datatypes.Container_Utility_File_Attachment) (resp datatypes.Ticket_Attachment_File, err error) {
	params := []interface{}{
		fileAttachment,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addAttachedFile", params, &r.Options, &resp)
	return
}

// Attach the given hardware to a SoftLayer ticket. A hardware attachment provides an easy way for SoftLayer's employees to quickly look up your hardware records in the case of hardware-specific issues.
func (r Ticket) AddAttachedHardware(hardwareId *int) (resp datatypes.Ticket_Attachment_Hardware, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addAttachedHardware", params, &r.Options, &resp)
	return
}

// Attach the given CloudLayer Computing Instance to a SoftLayer ticket. An attachment provides an easy way for SoftLayer's employees to quickly look up your records in the case of specific issues.
func (r Ticket) AddAttachedVirtualGuest(guestId *int) (resp datatypes.Ticket_Attachment_Virtual_Guest, err error) {
	params := []interface{}{
		guestId,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addAttachedVirtualGuest", params, &r.Options, &resp)
	return
}

// As part of the customer service process SoftLayer has provided a quick feedback mechanism for its customers to rate their overall experience with SoftLayer after a ticket is closed. addFinalComments() sets these comments for a ticket update made by a SoftLayer employee. Final comments may only be set on closed tickets, can only be set once, and may not exceed 4000 characters in length. Once the comments are set ''addFinalComments()'' returns a boolean true.
func (r Ticket) AddFinalComments(finalComments *string) (resp bool, err error) {
	params := []interface{}{
		finalComments,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addFinalComments", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Ticket) AddScheduledAlert(activationTime *string) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		activationTime,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addScheduledAlert", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Ticket) AddScheduledAutoClose(activationTime *string) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		activationTime,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addScheduledAutoClose", params, &r.Options, &resp)
	return
}

// Add an update to a ticket. A ticket update's entry has a maximum length of 4000 characters, so ''addUpdate()'' splits the ''entry'' property in the ''templateObject'' parameter into 3900 character blocks and creates one entry per 3900 character block. Once complete ''addUpdate()'' emails the ticket's owner and additional email addresses with an update message if the ticket's ''notifyUserOnUpdateFlag'' is set. If the ticket is a Legal or Abuse ticket, then the account's abuse emails are also notified when the updates are processed. Finally, ''addUpdate()'' returns an array of the newly created ticket updates.
func (r Ticket) AddUpdate(templateObject *datatypes.Ticket_Update, attachedFiles []datatypes.Container_Utility_File_Attachment) (resp []datatypes.Ticket_Update, err error) {
	params := []interface{}{
		templateObject,
		attachedFiles,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "addUpdate", params, &r.Options, &resp)
	return
}

// Create an administrative support ticket. Use an administrative ticket if you require SoftLayer's assistance managing your server or content. If you are experiencing an issue with SoftLayer's hardware, network, or services then please open a standard support ticket.
//
// Support tickets may only be created in the open state. The SoftLayer API defaults new ticket properties ''userEditableFlag'' to true, ''accountId'' to the id of the account that your API user belongs to, and ''statusId'' to 1001 (or "open"). You may not assign your new to ticket to users that your API user does not have access to.
//
// Once your ticket is created it is placed in a queue for SoftLayer employees to work. As they update the ticket new [[SoftLayer_Ticket_Update]] entries are added to the ticket object.
//
// Administrative support tickets add a one-time $3USD charge to your account.
func (r Ticket) CreateAdministrativeTicket(templateObject *datatypes.Ticket, contents *string, attachmentId *int, rootPassword *string, controlPanelPassword *string, accessPort *string, attachedFiles []datatypes.Container_Utility_File_Attachment, attachmentType *string) (resp datatypes.Ticket, err error) {
	params := []interface{}{
		templateObject,
		contents,
		attachmentId,
		rootPassword,
		controlPanelPassword,
		accessPort,
		attachedFiles,
		attachmentType,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "createAdministrativeTicket", params, &r.Options, &resp)
	return
}

// A cancel server request creates a ticket to cancel the resource on next bill date. The hardware ID parameter is required to determine which server is to be cancelled. NOTE: Hourly bare metal servers will be cancelled on next bill date.
//
// The reason parameter could be from the list below:
// * "No longer needed"
// * "Business closing down"
// * "Server / Upgrade Costs"
// * "Migrating to larger server"
// * "Migrating to smaller server"
// * "Migrating to a different SoftLayer datacenter"
// * "Network performance / latency"
// * "Support response / timing"
// * "Sales process / upgrades"
// * "Moving to competitor"
//
//
// The content parameter describes further the reason for cancelling the server.
func (r Ticket) CreateCancelServerTicket(attachmentId *int, reason *string, content *string, cancelAssociatedItems *bool, attachmentType *string) (resp datatypes.Ticket, err error) {
	params := []interface{}{
		attachmentId,
		reason,
		content,
		cancelAssociatedItems,
		attachmentType,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "createCancelServerTicket", params, &r.Options, &resp)
	return
}

// A cancel service request creates a sales ticket. The hardware ID parameter is required to determine which server is to be cancelled.
//
// The reason parameter could be from the list below:
// * "No longer needed"
// * "Business closing down"
// * "Server / Upgrade Costs"
// * "Migrating to larger server"
// * "Migrating to smaller server"
// * "Migrating to a different SoftLayer datacenter"
// * "Network performance / latency"
// * "Support response / timing"
// * "Sales process / upgrades"
// * "Moving to competitor"
//
//
// The content parameter describes further the reason for cancelling service.
func (r Ticket) CreateCancelServiceTicket(attachmentId *int, reason *string, content *string, attachmentType *string) (resp datatypes.Ticket, err error) {
	params := []interface{}{
		attachmentId,
		reason,
		content,
		attachmentType,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "createCancelServiceTicket", params, &r.Options, &resp)
	return
}

// Create a standard support ticket. Use a standard support ticket if you need to work out a problem related to SoftLayer's hardware, network, or services. If you require SoftLayer's assistance managing your server or content then please open an administrative ticket.
//
// Support tickets may only be created in the open state. The SoftLayer API defaults new ticket properties ''userEditableFlag'' to true, ''accountId'' to the id of the account that your API user belongs to, and ''statusId'' to 1001 (or "open"). You may not assign your new to ticket to users that your API user does not have access to.
//
// Once your ticket is created it is placed in a queue for SoftLayer employees to work. As they update the ticket new [[SoftLayer_Ticket_Update]] entries are added to the ticket object.
func (r Ticket) CreateStandardTicket(templateObject *datatypes.Ticket, contents *string, attachmentId *int, rootPassword *string, controlPanelPassword *string, accessPort *string, attachedFiles []datatypes.Container_Utility_File_Attachment, attachmentType *string) (resp datatypes.Ticket, err error) {
	params := []interface{}{
		templateObject,
		contents,
		attachmentId,
		rootPassword,
		controlPanelPassword,
		accessPort,
		attachedFiles,
		attachmentType,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "createStandardTicket", params, &r.Options, &resp)
	return
}

// Create a ticket for the SoftLayer sales team to perform a hardware or service upgrade. Our sales team will work with you on upgrade feasibility and pricing and then send the upgrade ticket to the proper department to perform the actual upgrade. Service affecting upgrades, such as server hardware or CloudLayer Computing Instance upgrades that require the server powered down must have a two hour maintenance specified for our datacenter engineers to perform your upgrade. Account level upgrades, such as adding PPTP VPN users, CDNLayer accounts, and monitoring services are processed much faster and do not require a maintenance window.
func (r Ticket) CreateUpgradeTicket(attachmentId *int, genericUpgrade *string, upgradeMaintenanceWindow *string, details *string, attachmentType *string, title *string) (resp datatypes.Ticket, err error) {
	params := []interface{}{
		attachmentId,
		genericUpgrade,
		upgradeMaintenanceWindow,
		details,
		attachmentType,
		title,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "createUpgradeTicket", params, &r.Options, &resp)
	return
}

// Edit a SoftLayer ticket. The edit method is two-fold. You may either edit a ticket itself, add an update to a ticket, attach up to two files to a ticket, or perform all of these tasks. The SoftLayer API ignores changes made to the ''userEditableFlag''  and ''accountId'' properties. You may not assign a ticket to a user that your API account does not have access to. You may not enter a custom title for standard support tickets, buy may do so when editing an administrative ticket. Finally, you may not close a ticket using this method. Please contact SoftLayer if you need a ticket closed.
//
// If you need to only add an update to a ticket then please use the [[SoftLayer_Ticket::addUpdate|addUpdate]] method in this service. Likewise if you need to only attach a file to a ticket then use the [[SoftLayer_Ticket::addAttachedFile|addAttachedFile]] method. The edit method exists as a convenience if you need to perform all these tasks at once.
func (r Ticket) Edit(templateObject *datatypes.Ticket, contents *string, attachedFiles []datatypes.Container_Utility_File_Attachment) (resp datatypes.Ticket, err error) {
	params := []interface{}{
		templateObject,
		contents,
		attachedFiles,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "edit", params, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer customer account associated with a ticket.
func (r Ticket) GetAccount() (resp datatypes.Account, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAccount", nil, &r.Options, &resp)
	return
}

// getAllTicketGroups() retrieves a list of all groups that a ticket may be assigned to. Ticket groups represent the internal department at SoftLayer who a ticket is assigned to.
//
// Every SoftLayer ticket has groupId and ticketGroup properties that correspond to one of the groups returned by getAllTicketGroups().
func (r Ticket) GetAllTicketGroups() (resp []datatypes.Ticket_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAllTicketGroups", nil, &r.Options, &resp)
	return
}

// getAllTicketStatuses() retrieves a list of all statuses that a ticket may exist in. Ticket status represent the current state of a ticket, usually "open", "assigned", and "closed".
//
// Every SoftLayer ticket has statusId and status properties that correspond to one of the statuses returned by getAllTicketStatuses().
func (r Ticket) GetAllTicketStatuses() (resp []datatypes.Ticket_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAllTicketStatuses", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetAssignedAgents() (resp []datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAssignedAgents", nil, &r.Options, &resp)
	return
}

// Retrieve The portal user that a ticket is assigned to.
func (r Ticket) GetAssignedUser() (resp datatypes.User_Customer, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAssignedUser", nil, &r.Options, &resp)
	return
}

// Retrieve The list of additional emails to notify when a ticket update is made.
func (r Ticket) GetAttachedAdditionalEmails() (resp []datatypes.User_Customer_AdditionalEmail, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAttachedAdditionalEmails", nil, &r.Options, &resp)
	return
}

// Retrieve the file attached to a SoftLayer ticket by it's given identifier. To retrieve a list of files attached to a ticket either call the SoftLayer_Ticket::getAttachedFiles method or call SoftLayer_Ticket::getObject with ''attachedFiles'' defined in an object mask.
func (r Ticket) GetAttachedFile(attachmentId *int) (resp []byte, err error) {
	params := []interface{}{
		attachmentId,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAttachedFile", params, &r.Options, &resp)
	return
}

// Retrieve The files attached to a ticket.
func (r Ticket) GetAttachedFiles() (resp []datatypes.Ticket_Attachment_File, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAttachedFiles", nil, &r.Options, &resp)
	return
}

// Retrieve The hardware associated with a ticket. This is used in cases where a ticket is directly associated with one or more pieces of hardware.
func (r Ticket) GetAttachedHardware() (resp []datatypes.Hardware, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAttachedHardware", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetAttachedHardwareCount() (resp uint, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAttachedHardwareCount", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetAttachedResources() (resp []datatypes.Ticket_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAttachedResources", nil, &r.Options, &resp)
	return
}

// Retrieve The virtual guests associated with a ticket. This is used in cases where a ticket is directly associated with one or more virtualized guests installations or Virtual Servers.
func (r Ticket) GetAttachedVirtualGuests() (resp []datatypes.Virtual_Guest, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAttachedVirtualGuests", nil, &r.Options, &resp)
	return
}

// Retrieve Ticket is waiting on a response from a customer flag.
func (r Ticket) GetAwaitingUserResponseFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getAwaitingUserResponseFlag", nil, &r.Options, &resp)
	return
}

// Retrieve A service cancellation request.
func (r Ticket) GetCancellationRequest() (resp datatypes.Billing_Item_Cancellation_Request, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getCancellationRequest", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetEmployeeAttachments() (resp []datatypes.User_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getEmployeeAttachments", nil, &r.Options, &resp)
	return
}

// Retrieve The first physical or virtual server attached to a ticket.
func (r Ticket) GetFirstAttachedResource() (resp datatypes.Ticket_Attachment, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getFirstAttachedResource", nil, &r.Options, &resp)
	return
}

// Retrieve The first update made to a ticket. This is typically the contents of a ticket when it's created.
func (r Ticket) GetFirstUpdate() (resp datatypes.Ticket_Update, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getFirstUpdate", nil, &r.Options, &resp)
	return
}

// Retrieve The SoftLayer department that a ticket is assigned to.
func (r Ticket) GetGroup() (resp datatypes.Ticket_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getGroup", nil, &r.Options, &resp)
	return
}

// Retrieve The invoice items associated with a ticket. Ticket based invoice items only exist when a ticket incurs a fee that has been invoiced.
func (r Ticket) GetInvoiceItems() (resp []datatypes.Billing_Invoice_Item, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getInvoiceItems", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetLastActivity() (resp datatypes.Ticket_Activity, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getLastActivity", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetLastEditor() (resp datatypes.User_Interface, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getLastEditor", nil, &r.Options, &resp)
	return
}

// Retrieve The last update made to a ticket.
func (r Ticket) GetLastUpdate() (resp datatypes.Ticket_Update, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getLastUpdate", nil, &r.Options, &resp)
	return
}

// Retrieve A timestamp of the last time the Ticket was viewed by the active user.
func (r Ticket) GetLastViewedDate() (resp datatypes.Time, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getLastViewedDate", nil, &r.Options, &resp)
	return
}

// Retrieve A ticket's associated location within the SoftLayer location hierarchy.
func (r Ticket) GetLocation() (resp datatypes.Location, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getLocation", nil, &r.Options, &resp)
	return
}

// Retrieve True if there are new, unread updates to this ticket for the current user, False otherwise.
func (r Ticket) GetNewUpdatesFlag() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getNewUpdatesFlag", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Ticket object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Ticket service. You can only retrieve tickets that are associated with your SoftLayer customer account.
func (r Ticket) GetObject() (resp datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetScheduledActions() (resp []datatypes.Provisioning_Version1_Transaction, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getScheduledActions", nil, &r.Options, &resp)
	return
}

// Retrieve The invoice associated with a ticket. Only tickets with an associated administrative charge have an invoice.
func (r Ticket) GetServerAdministrationBillingInvoice() (resp datatypes.Billing_Invoice, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getServerAdministrationBillingInvoice", nil, &r.Options, &resp)
	return
}

// Retrieve The refund invoice associated with a ticket. Only tickets with a refund applied in them have an associated refund invoice.
func (r Ticket) GetServerAdministrationRefundInvoice() (resp datatypes.Billing_Invoice, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getServerAdministrationRefundInvoice", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetServiceProvider() (resp datatypes.Service_Provider, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getServiceProvider", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetState() (resp []datatypes.Ticket_State, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getState", nil, &r.Options, &resp)
	return
}

// Retrieve A ticket's status.
func (r Ticket) GetStatus() (resp datatypes.Ticket_Status, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getStatus", nil, &r.Options, &resp)
	return
}

// Retrieve A ticket's subject. Only standard support tickets have an associated subject. A standard support ticket's title corresponds with it's subject's name.
func (r Ticket) GetSubject() (resp datatypes.Ticket_Subject, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getSubject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket) GetTagReferences() (resp []datatypes.Tag_Reference, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getTagReferences", nil, &r.Options, &resp)
	return
}

// Retrieve all tickets closed since a given date.
func (r Ticket) GetTicketsClosedSinceDate(closeDate *datatypes.Time) (resp []datatypes.Ticket, err error) {
	params := []interface{}{
		closeDate,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "getTicketsClosedSinceDate", params, &r.Options, &resp)
	return
}

// Retrieve A ticket's updates.
func (r Ticket) GetUpdates() (resp []datatypes.Ticket_Update, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "getUpdates", nil, &r.Options, &resp)
	return
}

// Mark a ticket as viewed.  All currently posted updates will be marked as viewed. The lastViewedDate property will be updated to the current time.
func (r Ticket) MarkAsViewed() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Ticket", "markAsViewed", nil, &r.Options, &resp)
	return
}

//
//
//
func (r Ticket) RemoveAssignedAgent(agentId *int) (err error) {
	var resp datatypes.Void
	params := []interface{}{
		agentId,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "removeAssignedAgent", params, &r.Options, &resp)
	return
}

// removeAttachedAdditionalEmails() removes the specified email addresses from a ticket's notification list. If one of the provided email addresses is not attached to the ticket then ''removeAttachedAdditiaonalEmails()'' ignores it and continues to the next one. Once the email addresses are removed ''removeAttachedAdditiaonalEmails()'' returns a boolean true.
func (r Ticket) RemoveAttachedAdditionalEmails(emails []string) (resp bool, err error) {
	params := []interface{}{
		emails,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "removeAttachedAdditionalEmails", params, &r.Options, &resp)
	return
}

// detach the given hardware from a SoftLayer ticket. Removing a hardware attachment may delay ticket processing time if the hardware removed is relevant to the ticket's issue. Return a boolean true upon successful hardware detachment.
func (r Ticket) RemoveAttachedHardware(hardwareId *int) (resp bool, err error) {
	params := []interface{}{
		hardwareId,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "removeAttachedHardware", params, &r.Options, &resp)
	return
}

// Detach the given CloudLayer Computing Instance from a SoftLayer ticket. Removing an attachment may delay ticket processing time if the instance removed is relevant to the ticket's issue. Return a boolean true upon successful detachment.
func (r Ticket) RemoveAttachedVirtualGuest(guestId *int) (resp bool, err error) {
	params := []interface{}{
		guestId,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "removeAttachedVirtualGuest", params, &r.Options, &resp)
	return
}

// no documentation yet
func (r Ticket) RemoveScheduledAlert() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Ticket", "removeScheduledAlert", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Ticket) RemoveScheduledAutoClose() (err error) {
	var resp datatypes.Void
	err = r.Session.DoRequest("SoftLayer_Ticket", "removeScheduledAutoClose", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Ticket) SetTags(tags *string) (resp bool, err error) {
	params := []interface{}{
		tags,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "setTags", params, &r.Options, &resp)
	return
}

// (DEPRECATED) Use [[SoftLayer_Ticket_Survey::getPreference]] method.
func (r Ticket) SurveyEligible() (resp bool, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket", "surveyEligible", nil, &r.Options, &resp)
	return
}

// Creates new additional emails for assigned user if new emails are provided. Attaches any newly created additional emails to ticket. Remove any additional emails from a ticket that are not provided as part of $emails
func (r Ticket) UpdateAttachedAdditionalEmails(emails []string) (resp bool, err error) {
	params := []interface{}{
		emails,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket", "updateAttachedAdditionalEmails", params, &r.Options, &resp)
	return
}

// SoftLayer tickets can have have files attached to them. Attaching a file to a ticket is a good way to report issues, provide documentation, and give examples of an issue. Both SoftLayer customers and employees have the ability to attach files to a ticket. The SoftLayer_Ticket_Attachment_File data type models a single file attached to a ticket.
type Ticket_Attachment_File struct {
	Session *session.Session
	Options sl.Options
}

// GetTicketAttachmentFileService returns an instance of the Ticket_Attachment_File SoftLayer service
func GetTicketAttachmentFileService(sess *session.Session) Ticket_Attachment_File {
	return Ticket_Attachment_File{Session: sess}
}

func (r Ticket_Attachment_File) Id(id int) Ticket_Attachment_File {
	r.Options.Id = &id
	return r
}

func (r Ticket_Attachment_File) Mask(mask string) Ticket_Attachment_File {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Ticket_Attachment_File) Filter(filter string) Ticket_Attachment_File {
	r.Options.Filter = filter
	return r
}

func (r Ticket_Attachment_File) Limit(limit int) Ticket_Attachment_File {
	r.Options.Limit = &limit
	return r
}

func (r Ticket_Attachment_File) Offset(offset int) Ticket_Attachment_File {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Ticket_Attachment_File) GetExtensionWhitelist() (resp []string, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Attachment_File", "getExtensionWhitelist", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Ticket_Attachment_File) GetObject() (resp datatypes.Ticket_Attachment_File, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Attachment_File", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket_Attachment_File) GetTicket() (resp datatypes.Ticket, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Attachment_File", "getTicket", nil, &r.Options, &resp)
	return
}

// Retrieve The ticket that a file is attached to.
func (r Ticket_Attachment_File) GetUpdate() (resp datatypes.Ticket_Update, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Attachment_File", "getUpdate", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Ticket_Priority struct {
	Session *session.Session
	Options sl.Options
}

// GetTicketPriorityService returns an instance of the Ticket_Priority SoftLayer service
func GetTicketPriorityService(sess *session.Session) Ticket_Priority {
	return Ticket_Priority{Session: sess}
}

func (r Ticket_Priority) Id(id int) Ticket_Priority {
	r.Options.Id = &id
	return r
}

func (r Ticket_Priority) Mask(mask string) Ticket_Priority {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Ticket_Priority) Filter(filter string) Ticket_Priority {
	r.Options.Filter = filter
	return r
}

func (r Ticket_Priority) Limit(limit int) Ticket_Priority {
	r.Options.Limit = &limit
	return r
}

func (r Ticket_Priority) Offset(offset int) Ticket_Priority {
	r.Options.Offset = &offset
	return r
}

// no documentation yet
func (r Ticket_Priority) GetPriorities() (resp []datatypes.Container_Ticket_Priority, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Priority", "getPriorities", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Ticket_Subject data type models one of the possible subjects that a standard support ticket may belong to. A basic support ticket's title matches it's corresponding subject's name.
type Ticket_Subject struct {
	Session *session.Session
	Options sl.Options
}

// GetTicketSubjectService returns an instance of the Ticket_Subject SoftLayer service
func GetTicketSubjectService(sess *session.Session) Ticket_Subject {
	return Ticket_Subject{Session: sess}
}

func (r Ticket_Subject) Id(id int) Ticket_Subject {
	r.Options.Id = &id
	return r
}

func (r Ticket_Subject) Mask(mask string) Ticket_Subject {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Ticket_Subject) Filter(filter string) Ticket_Subject {
	r.Options.Filter = filter
	return r
}

func (r Ticket_Subject) Limit(limit int) Ticket_Subject {
	r.Options.Limit = &limit
	return r
}

func (r Ticket_Subject) Offset(offset int) Ticket_Subject {
	r.Options.Offset = &offset
	return r
}

// Retrieve all possible ticket subjects. The SoftLayer customer portal uses this method in the add standard support ticket form.
func (r Ticket_Subject) GetAllObjects() (resp []datatypes.Ticket_Subject, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject", "getAllObjects", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket_Subject) GetCategory() (resp datatypes.Ticket_Subject_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject", "getCategory", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket_Subject) GetGroup() (resp datatypes.Ticket_Group, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject", "getGroup", nil, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Ticket_Subject object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Ticket_Subject service.
func (r Ticket_Subject) GetObject() (resp datatypes.Ticket_Subject, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject", "getObject", nil, &r.Options, &resp)
	return
}

// SoftLayer maintains relationships between the generic subjects for standard administration and the top five commonly asked questions about these subjects. getTopFileKnowledgeLayerQuestions() retrieves the top five questions and answers from the SoftLayer KnowledgeLayer related to the given ticket subject.
func (r Ticket_Subject) GetTopFiveKnowledgeLayerQuestions() (resp []datatypes.Container_KnowledgeLayer_QuestionAnswer, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject", "getTopFiveKnowledgeLayerQuestions", nil, &r.Options, &resp)
	return
}

// SoftLayer_Ticket_Subject_Category groups ticket subjects into logical group.
type Ticket_Subject_Category struct {
	Session *session.Session
	Options sl.Options
}

// GetTicketSubjectCategoryService returns an instance of the Ticket_Subject_Category SoftLayer service
func GetTicketSubjectCategoryService(sess *session.Session) Ticket_Subject_Category {
	return Ticket_Subject_Category{Session: sess}
}

func (r Ticket_Subject_Category) Id(id int) Ticket_Subject_Category {
	r.Options.Id = &id
	return r
}

func (r Ticket_Subject_Category) Mask(mask string) Ticket_Subject_Category {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Ticket_Subject_Category) Filter(filter string) Ticket_Subject_Category {
	r.Options.Filter = filter
	return r
}

func (r Ticket_Subject_Category) Limit(limit int) Ticket_Subject_Category {
	r.Options.Limit = &limit
	return r
}

func (r Ticket_Subject_Category) Offset(offset int) Ticket_Subject_Category {
	r.Options.Offset = &offset
	return r
}

// Retrieve all ticket subject categories.
func (r Ticket_Subject_Category) GetAllObjects() (resp []datatypes.Ticket_Subject_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject_Category", "getAllObjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
func (r Ticket_Subject_Category) GetObject() (resp datatypes.Ticket_Subject_Category, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject_Category", "getObject", nil, &r.Options, &resp)
	return
}

// Retrieve
func (r Ticket_Subject_Category) GetSubjects() (resp []datatypes.Ticket_Subject, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Subject_Category", "getSubjects", nil, &r.Options, &resp)
	return
}

// no documentation yet
type Ticket_Survey struct {
	Session *session.Session
	Options sl.Options
}

// GetTicketSurveyService returns an instance of the Ticket_Survey SoftLayer service
func GetTicketSurveyService(sess *session.Session) Ticket_Survey {
	return Ticket_Survey{Session: sess}
}

func (r Ticket_Survey) Id(id int) Ticket_Survey {
	r.Options.Id = &id
	return r
}

func (r Ticket_Survey) Mask(mask string) Ticket_Survey {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Ticket_Survey) Filter(filter string) Ticket_Survey {
	r.Options.Filter = filter
	return r
}

func (r Ticket_Survey) Limit(limit int) Ticket_Survey {
	r.Options.Limit = &limit
	return r
}

func (r Ticket_Survey) Offset(offset int) Ticket_Survey {
	r.Options.Offset = &offset
	return r
}

// Use this method to retrieve the ticket survey preferences. It will return your [[SoftLayer_Container_Ticket_Survey_Preference|survey preference]] which indicates if your account is applicable to receive a survey and if you're opted in. You can control the survey opt via the [[SoftLayer_Ticket_Survey::optIn|opt-in]] or [[SoftLayer_Ticket_Survey::optOut|opt-out]] method.
func (r Ticket_Survey) GetPreference() (resp datatypes.Container_Ticket_Survey_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Survey", "getPreference", nil, &r.Options, &resp)
	return
}

// You will not receive a ticket survey if you are opted out. Use this method to opt back in if you wish to provide feedback to our support team. You may use the [[SoftLayer_Ticket_Survey::getPreference|getPreference]] method to check your current opt status.
//
// This method is depricated. Use [[SoftLayer_User_Customer::changePreference]] instead.
func (r Ticket_Survey) OptIn() (resp datatypes.Container_Ticket_Survey_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Survey", "optIn", nil, &r.Options, &resp)
	return
}

// By default, customers will occasionally receive a ticket survey upon closing of a ticket. Use this method to opt out of it for the next 90 days. Ticket surveys may not be applicable for some customers. Use the [[SoftLayer_Ticket_Survey::getPreference|getPreference]] method to retrieve your survey preference. The "applicable" property of the [[SoftLayer_Container_Ticket_Survey_Preference|survey preference]] indicates if the survey is relevant to your account or not.
//
// This method is depricated. Use [[SoftLayer_User_Customer::changePreference]] instead.
func (r Ticket_Survey) OptOut() (resp datatypes.Container_Ticket_Survey_Preference, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Survey", "optOut", nil, &r.Options, &resp)
	return
}

// The SoftLayer_Ticket_Update_Employee data type models an update to a ticket made by a SoftLayer employee.
type Ticket_Update_Employee struct {
	Session *session.Session
	Options sl.Options
}

// GetTicketUpdateEmployeeService returns an instance of the Ticket_Update_Employee SoftLayer service
func GetTicketUpdateEmployeeService(sess *session.Session) Ticket_Update_Employee {
	return Ticket_Update_Employee{Session: sess}
}

func (r Ticket_Update_Employee) Id(id int) Ticket_Update_Employee {
	r.Options.Id = &id
	return r
}

func (r Ticket_Update_Employee) Mask(mask string) Ticket_Update_Employee {
	if !strings.HasPrefix(mask, "mask[") && (strings.Contains(mask, "[") || strings.Contains(mask, ",")) {
		mask = fmt.Sprintf("mask[%s]", mask)
	}

	r.Options.Mask = mask
	return r
}

func (r Ticket_Update_Employee) Filter(filter string) Ticket_Update_Employee {
	r.Options.Filter = filter
	return r
}

func (r Ticket_Update_Employee) Limit(limit int) Ticket_Update_Employee {
	r.Options.Limit = &limit
	return r
}

func (r Ticket_Update_Employee) Offset(offset int) Ticket_Update_Employee {
	r.Options.Offset = &offset
	return r
}

// As part of the customer service process SoftLayer has provided a quick feedback mechanism for its customers to rate the responses that its employees give on tickets. addResponseRating() sets the rating for a single ticket update made by a SoftLayer employee. Ticket ratings have the integer values 1 through 5, with 1 being the worst and 5 being the best. Once the rating is set ''addResponseRating()'' returns a boolean true.
func (r Ticket_Update_Employee) AddResponseRating(responseRating *int) (resp bool, err error) {
	params := []interface{}{
		responseRating,
	}
	err = r.Session.DoRequest("SoftLayer_Ticket_Update_Employee", "addResponseRating", params, &r.Options, &resp)
	return
}

// getObject retrieves the SoftLayer_Ticket_Update_Employee object whose ID number corresponds to the ID number of the init parameter passed to the SoftLayer_Ticket_Update_Employee service. You can only retrieve employee updates to tickets that your API account has access to.
func (r Ticket_Update_Employee) GetObject() (resp datatypes.Ticket_Update_Employee, err error) {
	err = r.Session.DoRequest("SoftLayer_Ticket_Update_Employee", "getObject", nil, &r.Options, &resp)
	return
}
