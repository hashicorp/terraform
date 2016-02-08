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
	Id        int    `json:"id"`
	RelatedId int    `json:"related_event_id"`
	Handle    string `json:"handle"`
	Message   string `json:"message"`
	Resource  string `json:"resource"`
	Url       string `json:"url"`
}

// reqComment is the container for receiving commenst.
type reqComment struct {
	Comment Comment `json:"comment"`
}

// CreateComment adds a new comment to the system.
func (self *Client) CreateComment(handle, message string) (*Comment, error) {
	var out reqComment
	comment := Comment{Handle: handle, Message: message}
	err := self.doJsonRequest("POST", "/v1/comments", &comment, &out)
	if err != nil {
		return nil, err
	}
	return &out.Comment, nil
}

// CreateRelatedComment adds a new comment, but lets you specify the related
// identifier for the comment.
func (self *Client) CreateRelatedComment(handle, message string,
	relid int) (*Comment, error) {
	var out reqComment
	comment := Comment{Handle: handle, Message: message, RelatedId: relid}
	err := self.doJsonRequest("POST", "/v1/comments", &comment, &out)
	if err != nil {
		return nil, err
	}
	return &out.Comment, nil
}

// EditComment changes the message and possibly handle of a particular comment.
func (self *Client) EditComment(id int, handle, message string) error {
	comment := Comment{Handle: handle, Message: message}
	return self.doJsonRequest("PUT", fmt.Sprintf("/v1/comments/%d", id),
		&comment, nil)
}

// DeleteComment does exactly what you expect.
func (self *Client) DeleteComment(id int) error {
	return self.doJsonRequest("DELETE", fmt.Sprintf("/v1/comments/%d", id),
		nil, nil)
}
