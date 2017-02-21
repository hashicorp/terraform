/*
Copyright 2016. All rights reserved.
Use of this source code is governed by a Apache Software
license that can be found in the LICENSE file.
*/

//Package user provides requests and response structures to achieve User API actions.
package user

// CreateUserRequest provides necessary parameter structure for creating User
type CreateUserRequest struct {
	APIKey string `json:"apiKey,omitempty"`
	Username string `json:"username,omitempty"`
	Fullname string `json:"fullname,omitempty"`
	Role string `json:"role,omitempty"`
	Locale string `json:"locale,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// UpdateUserRequest provides necessary parameter structure for updating an User
type UpdateUserRequest struct {
	Id string `json:"id,omitempty"`
	APIKey string `json:"apiKey,omitempty"`
	Fullname string `json:"fullname,omitempty"`
	Role string `json:"role,omitempty"`
	Locale string `json:"locale,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// DeleteUserRequest provides necessary parameter structure for deleting an User
type DeleteUserRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id string `url:"id,omitempty"`
	Username string `url:"username,omitempty"`
}

// GetUserRequest provides necessary parameter structure for requesting User information
type GetUserRequest struct {
	APIKey string `url:"apiKey,omitempty"`
	Id string `url:"id,omitempty"`
	Username string `url:"username,omitempty"`
}

// ListUserRequest provides necessary parameter structure for listing Users
type ListUsersRequest struct {
	APIKey string `url:"apiKey,omitempty"`
}
