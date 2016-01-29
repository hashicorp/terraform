/*
Copyright (c) 2014-2015 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flags

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/vmware/govmomi/object"
	"golang.org/x/net/context"
)

type DatastoreFlag struct {
	common

	*DatacenterFlag

	name string
	ds   *object.Datastore
}

var datastoreFlagKey = flagKey("datastore")

func NewDatastoreFlag(ctx context.Context) (*DatastoreFlag, context.Context) {
	if v := ctx.Value(datastoreFlagKey); v != nil {
		return v.(*DatastoreFlag), ctx
	}

	v := &DatastoreFlag{}
	v.DatacenterFlag, ctx = NewDatacenterFlag(ctx)
	ctx = context.WithValue(ctx, datastoreFlagKey, v)
	return v, ctx
}

func (flag *DatastoreFlag) Register(ctx context.Context, f *flag.FlagSet) {
	flag.RegisterOnce(func() {
		flag.DatacenterFlag.Register(ctx, f)

		env := "GOVC_DATASTORE"
		value := os.Getenv(env)
		usage := fmt.Sprintf("Datastore [%s]", env)
		f.StringVar(&flag.name, "ds", value, usage)
	})
}

func (flag *DatastoreFlag) Process(ctx context.Context) error {
	return flag.ProcessOnce(func() error {
		if err := flag.DatacenterFlag.Process(ctx); err != nil {
			return err
		}
		return nil
	})
}

func (flag *DatastoreFlag) Datastore() (*object.Datastore, error) {
	if flag.ds != nil {
		return flag.ds, nil
	}

	finder, err := flag.Finder()
	if err != nil {
		return nil, err
	}

	if flag.ds, err = finder.DatastoreOrDefault(context.TODO(), flag.name); err != nil {
		return nil, err
	}

	return flag.ds, nil
}

func (flag *DatastoreFlag) DatastorePath(name string) (string, error) {
	ds, err := flag.Datastore()
	if err != nil {
		return "", err
	}

	return ds.Path(name), nil
}

func (flag *DatastoreFlag) DatastoreURL(path string) (*url.URL, error) {
	dc, err := flag.Datacenter()
	if err != nil {
		return nil, err
	}

	ds, err := flag.Datastore()
	if err != nil {
		return nil, err
	}

	u, err := ds.URL(context.TODO(), dc, path)
	if err != nil {
		return nil, err
	}

	return u, nil
}
