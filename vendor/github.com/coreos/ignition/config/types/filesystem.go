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

import (
	"encoding/json"
	"errors"
)

var (
	ErrFilesystemInvalidFormat = errors.New("invalid filesystem format")
	ErrFilesystemNoMountPath   = errors.New("filesystem is missing mount or path")
	ErrFilesystemMountAndPath  = errors.New("filesystem has both mount and path defined")
)

type Filesystem struct {
	Name  string           `json:"name,omitempty"  yaml:"name"`
	Mount *FilesystemMount `json:"mount,omitempty" yaml:"mount"`
	Path  Path             `json:"path,omitempty"  yaml:"path"`
}

type FilesystemMount struct {
	Device Path              `json:"device,omitempty" yaml:"device"`
	Format FilesystemFormat  `json:"format,omitempty" yaml:"format"`
	Create *FilesystemCreate `json:"create,omitempty" yaml:"create"`
}

type FilesystemCreate struct {
	Force   bool        `json:"force,omitempty"   yaml:"force"`
	Options MkfsOptions `json:"options,omitempty" yaml:"options"`
}

func (f *Filesystem) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return f.unmarshal(unmarshal)
}

func (f *Filesystem) UnmarshalJSON(data []byte) error {
	return f.unmarshal(func(tf interface{}) error {
		return json.Unmarshal(data, tf)
	})
}

type filesystem Filesystem

func (f *Filesystem) unmarshal(unmarshal func(interface{}) error) error {
	tf := filesystem(*f)
	if err := unmarshal(&tf); err != nil {
		return err
	}
	*f = Filesystem(tf)
	return f.assertValid()
}

func (f Filesystem) assertValid() error {
	hasMount := false
	hasPath := false

	if f.Mount != nil {
		hasMount = true
		if err := f.Mount.assertValid(); err != nil {
			return err
		}
	}

	if len(f.Path) != 0 {
		hasPath = true
		if err := f.Path.assertValid(); err != nil {
			return err
		}
	}

	if !hasMount && !hasPath {
		return ErrFilesystemNoMountPath
	} else if hasMount && hasPath {
		return ErrFilesystemMountAndPath
	}

	return nil
}

func (f *FilesystemMount) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return f.unmarshal(unmarshal)
}

func (f *FilesystemMount) UnmarshalJSON(data []byte) error {
	return f.unmarshal(func(tf interface{}) error {
		return json.Unmarshal(data, tf)
	})
}

type filesystemMount FilesystemMount

func (f *FilesystemMount) unmarshal(unmarshal func(interface{}) error) error {
	tf := filesystemMount(*f)
	if err := unmarshal(&tf); err != nil {
		return err
	}
	*f = FilesystemMount(tf)
	return f.assertValid()
}

func (f FilesystemMount) assertValid() error {
	if err := f.Device.assertValid(); err != nil {
		return err
	}
	if err := f.Format.assertValid(); err != nil {
		return err
	}
	return nil
}

type FilesystemFormat string

func (f *FilesystemFormat) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return f.unmarshal(unmarshal)
}

func (f *FilesystemFormat) UnmarshalJSON(data []byte) error {
	return f.unmarshal(func(tf interface{}) error {
		return json.Unmarshal(data, tf)
	})
}

type filesystemFormat FilesystemFormat

func (f *FilesystemFormat) unmarshal(unmarshal func(interface{}) error) error {
	tf := filesystemFormat(*f)
	if err := unmarshal(&tf); err != nil {
		return err
	}
	*f = FilesystemFormat(tf)
	return f.assertValid()
}

func (f FilesystemFormat) assertValid() error {
	switch f {
	case "ext4", "btrfs", "xfs":
		return nil
	default:
		return ErrFilesystemInvalidFormat
	}
}

type MkfsOptions []string

func (o *MkfsOptions) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return o.unmarshal(unmarshal)
}

func (o *MkfsOptions) UnmarshalJSON(data []byte) error {
	return o.unmarshal(func(to interface{}) error {
		return json.Unmarshal(data, to)
	})
}

type mkfsOptions MkfsOptions

func (o *MkfsOptions) unmarshal(unmarshal func(interface{}) error) error {
	to := mkfsOptions(*o)
	if err := unmarshal(&to); err != nil {
		return err
	}
	*o = MkfsOptions(to)
	return o.assertValid()
}

func (o MkfsOptions) assertValid() error {
	return nil
}
