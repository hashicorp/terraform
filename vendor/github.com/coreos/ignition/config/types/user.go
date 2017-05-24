// Copyright 2016 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

type User struct {
	Name              string      `json:"name,omitempty"`
	PasswordHash      string      `json:"passwordHash,omitempty"`
	SSHAuthorizedKeys []string    `json:"sshAuthorizedKeys,omitempty"`
	Create            *UserCreate `json:"create,omitempty"`
}

type UserCreate struct {
	Uid          *uint    `json:"uid,omitempty"`
	GECOS        string   `json:"gecos,omitempty"`
	Homedir      string   `json:"homeDir,omitempty"`
	NoCreateHome bool     `json:"noCreateHome,omitempty"`
	PrimaryGroup string   `json:"primaryGroup,omitempty"`
	Groups       []string `json:"groups,omitempty"`
	NoUserGroup  bool     `json:"noUserGroup,omitempty"`
	System       bool     `json:"system,omitempty"`
	NoLogInit    bool     `json:"noLogInit,omitempty"`
	Shell        string   `json:"shell,omitempty"`
}
