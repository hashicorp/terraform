// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonchecks

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
)

type staticObjectAddr map[string]interface{}

func makeStaticObjectAddr(addr addrs.ConfigCheckable) staticObjectAddr {
	ret := map[string]interface{}{
		"to_display": addr.String(),
	}

	switch addr := addr.(type) {
	case addrs.ConfigResource:
		if kind := addr.CheckableKind(); kind != addrs.CheckableResource {
			// Something has gone very wrong
			panic(fmt.Sprintf("%T has CheckableKind %s", addr, kind))
		}

		ret["kind"] = "resource"
		switch addr.Resource.Mode {
		case addrs.ManagedResourceMode:
			ret["mode"] = "managed"
		case addrs.DataResourceMode:
			ret["mode"] = "data"
		default:
			panic(fmt.Sprintf("unsupported resource mode %#v", addr.Resource.Mode))
		}
		ret["type"] = addr.Resource.Type
		ret["name"] = addr.Resource.Name
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
	case addrs.ConfigOutputValue:
		if kind := addr.CheckableKind(); kind != addrs.CheckableOutputValue {
			// Something has gone very wrong
			panic(fmt.Sprintf("%T has CheckableKind %s", addr, kind))
		}

		ret["kind"] = "output_value"
		ret["name"] = addr.OutputValue.Name
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
	case addrs.ConfigCheck:
		if kind := addr.CheckableKind(); kind != addrs.CheckableCheck {
			// Something has gone very wrong
			panic(fmt.Sprintf("%T has CheckableKind %s", addr, kind))
		}

		ret["kind"] = "check"
		ret["name"] = addr.Check.Name
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
	case addrs.ConfigInputVariable:
		if kind := addr.CheckableKind(); kind != addrs.CheckableInputVariable {
			// Something has gone very wrong
			panic(fmt.Sprintf("%T has CheckableKind %s", addr, kind))
		}

		ret["kind"] = "var"
		ret["name"] = addr.Variable.Name
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
	default:
		panic(fmt.Sprintf("unsupported ConfigCheckable implementation %T", addr))
	}

	return ret
}

type dynamicObjectAddr map[string]interface{}

func makeDynamicObjectAddr(addr addrs.Checkable) dynamicObjectAddr {
	ret := map[string]interface{}{
		"to_display": addr.String(),
	}

	switch addr := addr.(type) {
	case addrs.AbsResourceInstance:
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
		if addr.Resource.Key != addrs.NoKey {
			ret["instance_key"] = addr.Resource.Key
		}
	case addrs.AbsOutputValue:
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
	case addrs.AbsCheck:
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
	case addrs.AbsInputVariableInstance:
		if !addr.Module.IsRoot() {
			ret["module"] = addr.Module.String()
		}
	default:
		panic(fmt.Sprintf("unsupported Checkable implementation %T", addr))
	}

	return ret
}
