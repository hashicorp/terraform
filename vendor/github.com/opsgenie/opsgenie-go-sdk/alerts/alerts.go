/*
Copyright 2015 OpsGenie. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package alerts provides requests and response structures to achieve Alert API actions.
package alerts

import (
	"os"
)

// AcknowledgeAlertRequest provides necessary parameter structure to Acknowledge an alert at OpsGenie.
type AcknowledgeAlertRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	User   string `json:"user,omitempty"`
	Note   string `json:"note,omitempty"`
	Source string `json:"source,omitempty"`
}

// AddNoteAlertRequest provides necessary parameter structure to Add Note to an alert at OpsGenie.
type AddNoteAlertRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Note   string `json:"note,omitempty"`
	User   string `json:"user,omitempty"`
	Source string `json:"source,omitempty"`
}

// AddRecipientAlertRequest provides necessary parameter structure to Add Recipient to an alert at OpsGenie.
type AddRecipientAlertRequest struct {
	APIKey    string `json:"apiKey,omitempty"`
	ID        string `json:"id,omitempty"`
	Alias     string `json:"alias,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	User      string `json:"user,omitempty"`
	Note      string `json:"note,omitempty"`
	Source    string `json:"source,omitempty"`
}

// AddTeamAlertRequest provides necessary parameter structure to Add Team to an alert at OpsGenie.
type AddTeamAlertRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Team   string `json:"team,omitempty"`
	User   string `json:"user,omitempty"`
	Note   string `json:"note,omitempty"`
	Source string `json:"source,omitempty"`
}

// AddTagsAlertRequest provides necessary parameter structure to Add Tags to an alert at OpsGenie.
type AddTagsAlertRequest struct {
	APIKey string   `json:"apiKey,omitempty"`
	ID     string   `json:"id,omitempty"`
	Alias  string   `json:"alias,omitempty"`
	Tags   []string `json:"tags,omitempty"`
	User   string   `json:"user,omitempty"`
	Note   string   `json:"note,omitempty"`
	Source string   `json:"source,omitempty"`
}

// AssignOwnerAlertRequest provides necessary parameter structure to Assign a User as Owner to an alert at OpsGenie.
type AssignOwnerAlertRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Owner  string `json:"owner,omitempty"`
	User   string `json:"user,omitempty"`
	Note   string `json:"note,omitempty"`
	Source string `json:"source,omitempty"`
}

// AttachFileAlertRequest provides necessary parameter structure to Attach File to an alert at OpsGenie.
type AttachFileAlertRequest struct {
	APIKey     string   `json:"apiKey,omitempty"`
	ID         string   `json:"id,omitempty"`
	Alias      string   `json:"alias,omitempty"`
	Attachment *os.File `json:"attachment,omitempty"`
	User       string   `json:"user,omitempty"`
	Source     string   `json:"source,omitempty"`
	IndexFile  string   `json:"indexFile,omitempty"`
	Note       string   `json:"note,omitempty"`
}

// CloseAlertRequest provides necessary parameter structure to Close an alert at OpsGenie.
type CloseAlertRequest struct {
	APIKey string   `json:"apiKey,omitempty"`
	ID     string   `json:"id,omitempty"`
	Alias  string   `json:"alias,omitempty"`
	User   string   `json:"user,omitempty"`
	Note   string   `json:"note,omitempty"`
	Notify []string `json:"notify,omitempty"`
	Source string   `json:"source,omitempty"`
}

// CreateAlertRequest provides necessary parameter structure to Create an alert at OpsGenie.
type CreateAlertRequest struct {
	APIKey      string            `json:"apiKey,omitempty"`
	Message     string            `json:"message,omitempty"`
	Teams       []string          `json:"teams,omitempty"`
	Alias       string            `json:"alias,omitempty"`
	Description string            `json:"description,omitempty"`
	Recipients  []string          `json:"recipients,omitempty"`
	Actions     []string          `json:"actions,omitempty"`
	Source      string            `json:"source,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Details     map[string]string `json:"details,omitempty"`
	Entity      string            `json:"entity,omitempty"`
	User        string            `json:"user,omitempty"`
	Note        string            `json:"note,omitempty"`
}

// DeleteAlertRequest provides necessary parameter structure to Delete an alert from OpsGenie.
type DeleteAlertRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	ID     string `url:"id,omitempty"`
	Alias  string `url:"alias,omitempty"`
	User   string `url:"user,omitempty"`
	Source string `url:"source,omitempty"`
}

// ExecuteActionAlertRequest provides necessary parameter structure to Execute Custom Actions on an alert at OpsGenie.
type ExecuteActionAlertRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	Action string `json:"action,omitempty"`
	User   string `json:"user,omitempty"`
	Source string `json:"source,omitempty"`
	Note   string `json:"note,omitempty"`
}

// GetAlertRequest provides necessary parameter structure to Retrieve an alert details from OpsGenie.
type GetAlertRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	ID     string `url:"id,omitempty"`
	Alias  string `url:"alias,omitempty"`
	TinyID string `url:"tinyId,omitempty"`
}

// ListAlertLogsRequest provides necessary parameter structure to Retrieve activity logs of an alert from OpsGenie.
type ListAlertLogsRequest struct {
	APIKey  string `url:"apiKey,omitempty"`
	ID      string `url:"id,omitempty"`
	Alias   string `url:"alias,omitempty"`
	Limit   uint64 `url:"limit,omitempty"`
	Order   string `url:"order,omitempty"`
	LastKey string `url:"lastKey,omitempty"`
}

// ListAlertNotesRequest provides necessary parameter structure to Retrieve notes of an alert from OpsGenie.
type ListAlertNotesRequest struct {
	APIKey  string `url:"apiKey,omitempty"`
	ID      string `url:"id,omitempty"`
	Alias   string `url:"alias,omitempty"`
	Limit   uint64 `url:"limit,omitempty"`
	Order   string `url:"order,omitempty"`
	LastKey string `url:"lastKey,omitempty"`
}

// ListAlertRecipientsRequest provides necessary parameter structure to Retrieve recipients of an alert from OpsGenie.
type ListAlertRecipientsRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	ID     string `url:"id,omitempty"`
	Alias  string `url:"alias,omitempty"`
}

// ListAlertsRequest provides necessary parameter structure to Retrieve alerts from OpsGenie.
type ListAlertsRequest struct {
	APIKey        string 	`url:"apiKey,omitempty"`
	CreatedAfter  uint64 	`url:"createdAfter,omitempty"`
	CreatedBefore uint64 	`url:"createdBefore,omitempty"`
	UpdatedAfter  uint64 	`url:"updatedAfter,omitempty"`
	UpdatedBefore uint64 	`url:"updatedBefore,omitempty"`
	Limit         uint64 	`url:"limit,omitempty"`
	Status        string 	`url:"status,omitempty"`
	SortBy        string 	`url:"sortBy,omitempty"`
	Order         string 	`url:"order,omitempty"`
	Teams         []string  `url:"teams,omitempty"`
	Tags          []string  `url:"tags,omitempty"`
	TagsOperator  string 	`url:"tagsOperator,omitempty"`
}

// CountAlertRequest counts the alerts at OpsGenie.
type CountAlertRequest struct {
	APIKey        string `url:"apiKey,omitempty"`
	CreatedAfter  uint64 `url:"createdAfter,omitempty"`
	CreatedBefore uint64 `url:"createdBefore,omitempty"`
	UpdatedAfter  uint64 `url:"updatedAfter,omitempty"`
	UpdatedBefore uint64 `url:"updatedBefore,omitempty"`
	Limit         uint64 `url:"limit,omitempty"`
	Status        string `url:"status,omitempty"`
	Tags          []string `url:"tags,omitempty"`
	TagsOperator  string `url:"tagsOperator,omitempty"`
}

// RenotifyAlertRequest provides necessary parameter structure to Re-notify recipients at OpsGenie.
type RenotifyAlertRequest struct {
	APIKey     string   `json:"apiKey,omitempty"`
	ID         string   `json:"id,omitempty"`
	Alias      string   `json:"alias,omitempty"`
	Recipients []string `json:"recipients,omitempty"`
	User       string   `json:"user,omitempty"`
	Note       string   `json:"note,omitempty"`
	Source     string   `json:"source,omitempty"`
}

// TakeOwnershipAlertRequest provides necessary parameter structure to Become the Owner of an alert at OpsGenie.
type TakeOwnershipAlertRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	User   string `json:"user,omitempty"`
	Note   string `json:"note,omitempty"`
	Source string `json:"source,omitempty"`
}

// UnAcknowledgeAlertRequest provides necessary parameter structure to Unacknowledge an alert at OpsGenie.
type UnAcknowledgeAlertRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias,omitempty"`
	User   string `json:"user,omitempty"`
	Note   string `json:"note,omitempty"`
	Source string `json:"source,omitempty"`
}

// SnoozeAlertRequest provides necessary parameter structure to Snooze an alert at OpsGenie.
type SnoozeAlertRequest struct {
	APIKey 		string `json:"apiKey,omitempty"`
	ID     		string `json:"id,omitempty"`
	Alias  		string `json:"alias,omitempty"`
	EndDate 	string `json:"endDate,omitempty"`
	User   		string `json:"user,omitempty"`
	Note  		string `json:"note,omitempty"`
	Source 		string `json:"source,omitempty"`
	TimeZone	string `json:"timezone,omitempty"`
}

// RemoveTagsAlertRequest provides necessary parameter structure to Remove Tags from an alert at OpsGenie.
type RemoveTagsAlertRequest struct {
	APIKey string   `url:"apiKey,omitempty"`
	ID     string   `url:"id,omitempty"`
	Alias  string   `url:"alias,omitempty"`
	Tags   []string	`url:"tags,omitempty"`
	User   string   `url:"user,omitempty"`
	Note   string   `url:"note,omitempty"`
	Source string   `url:"source,omitempty"`
}

// AddDetailsAlertRequest provides necessary parameter structure to Add Details to an alert at OpsGenie.
type AddDetailsAlertRequest struct {
	APIKey 	string   		`json:"apiKey,omitempty"`
	ID     	string   		`json:"id,omitempty"`
	Alias  	string   		`json:"alias,omitempty"`
	Details	map[string]string	`json:"details,omitempty"`
	User   	string   		`json:"user,omitempty"`
	Note   	string   		`json:"note,omitempty"`
	Source 	string   		`json:"source,omitempty"`
}

// RemoveDetailsAlertRequest provides necessary parameter structure to Remove Details from an alert at OpsGenie.
type RemoveDetailsAlertRequest struct {
	APIKey 	string   	`url:"apiKey,omitempty"`
	ID     	string   	`url:"id,omitempty"`
	Alias  	string   	`url:"alias,omitempty"`
	Keys	[]string	`url:"keys,omitempty"`
	User   	string   	`url:"user,omitempty"`
	Note   	string   	`url:"note,omitempty"`
	Source 	string   	`url:"source,omitempty"`
}

// EscalateToNextAlertRequest provides necessary parameter structure to Escalate To Next for and alert at OpsGenie.
type EscalateToNextAlertRequest struct {
	APIKey 		string  `json:"apiKey,omitempty"`
	ID     		string  `json:"id,omitempty"`
	Alias  		string  `json:"alias,omitempty"`
	EscalationID	string	`json:"escalationId,omitempty"`
	EscalationName	string	`json:"escalationName,omitempty"`
	User   		string  `json:"user,omitempty"`
	Note   		string  `json:"note,omitempty"`
	Source 		string  `json:"source,omitempty"`
}