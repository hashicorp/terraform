/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

import (
	"fmt"
)

// Comment is a special form of event that appears in a stream.
type Comment struct {
	Id        *int    `json:"id,omitempty"`
	RelatedId *int    `json:"related_event_id,omitempty"`
	Handle    *string `json:"handle,omitempty"`
	Message   *string `json:"message,omitempty"`
	Resource  *string `json:"resource,omitempty"`
	Url       *string `json:"url,omitempty"`
}

// reqComment is the container for receiving commenst.
type reqComment struct {
	Comment *Comment `json:"comment,omitempty"`
}

// CreateComment adds a new comment to the system.
func (client *Client) CreateComment(handle, message string) (*Comment, error) {
	var out reqComment
	comment := Comment{Handle: String(handle), Message: String(message)}
	if err := client.doJsonRequest("POST", "/v1/comments", &comment, &out); err != nil {
		return nil, err
	}
	return out.Comment, nil
}

// CreateRelatedComment adds a new comment, but lets you specify the related
// identifier for the comment.
func (client *Client) CreateRelatedComment(handle, message string,
	relid int) (*Comment, error) {
	var out reqComment
	comment := Comment{Handle: String(handle), Message: String(message), RelatedId: Int(relid)}
	if err := client.doJsonRequest("POST", "/v1/comments", &comment, &out); err != nil {
		return nil, err
	}
	return out.Comment, nil
}

// EditComment changes the message and possibly handle of a particular comment.
func (client *Client) EditComment(id int, handle, message string) error {
	comment := Comment{Handle: String(handle), Message: String(message)}
	return client.doJsonRequest("PUT", fmt.Sprintf("/v1/comments/%d", id),
		&comment, nil)
}

// DeleteComment does exactly what you expect.
func (client *Client) DeleteComment(id int) error {
	return client.doJsonRequest("DELETE", fmt.Sprintf("/v1/comments/%d", id),
		nil, nil)
}
