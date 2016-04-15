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
	Name              string      `json:"name,omitempty"              yaml:"name"`
	PasswordHash      string      `json:"passwordHash,omitempty"      yaml:"password_hash"`
	SSHAuthorizedKeys []string    `json:"sshAuthorizedKeys,omitempty" yaml:"ssh_authorized_keys"`
	Create            *UserCreate `json:"create,omitempty"            yaml:"create"`
}

type UserCreate struct {
	Uid          *uint    `json:"uid,omitempty"          yaml:"uid"`
	GECOS        string   `json:"gecos,omitempty"        yaml:"gecos"`
	Homedir      string   `json:"homeDir,omitempty"      yaml:"home_dir"`
	NoCreateHome bool     `json:"noCreateHome,omitempty" yaml:"no_create_home"`
	PrimaryGroup string   `json:"primaryGroup,omitempty" yaml:"primary_group"`
	Groups       []string `json:"groups,omitempty"       yaml:"groups"`
	NoUserGroup  bool     `json:"noUserGroup,omitempty"  yaml:"no_user_group"`
	System       bool     `json:"system,omitempty"       yaml:"system"`
	NoLogInit    bool     `json:"noLogInit,omitempty"    yaml:"no_log_init"`
	Shell        string   `json:"shell,omitempty"        yaml:"shell"`
}
