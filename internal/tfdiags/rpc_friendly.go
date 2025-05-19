// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package tfdiags

import (
	"encoding/gob"
)

type rpcFriendlyDiag struct {
	Severity_ Severity
	Summary_  string
	Detail_   string
	Subject_  *SourceRange
	Context_  *SourceRange
}

// rpcFriendlyDiag transforms a given diagnostic so that is more friendly to
// RPC.
//
// In particular, it currently returns an object that can be serialized and
// later re-inflated using gob. This definition may grow to include other
// serializations later.
func makeRPCFriendlyDiag(diag Diagnostic) Diagnostic {
	desc := diag.Description()
	source := diag.Source()
	return &rpcFriendlyDiag{
		Severity_: diag.Severity(),
		Summary_:  desc.Summary,
		Detail_:   desc.Detail,
		Subject_:  source.Subject,
		Context_:  source.Context,
	}
}

func (d *rpcFriendlyDiag) Severity() Severity {
	return d.Severity_
}

func (d *rpcFriendlyDiag) Description() Description {
	return Description{
		Summary: d.Summary_,
		Detail:  d.Detail_,
	}
}

func (d *rpcFriendlyDiag) Source() Source {
	return Source{
		Subject: d.Subject_,
		Context: d.Context_,
	}
}

func (d *rpcFriendlyDiag) Equals(otherDiag ComparableDiagnostic) bool {
	od, ok := otherDiag.(*rpcFriendlyDiag)
	if !ok {
		return false
	}
	if d.Severity_ != od.Severity_ {
		return false
	}
	if d.Summary_ != od.Summary_ {
		return false
	}
	if d.Detail_ != od.Detail_ {
		return false
	}
	if !sourceRangeEquals(d.Subject_, od.Subject_) {
		return false
	}

	return true
}

func (d rpcFriendlyDiag) FromExpr() *FromExpr {
	// RPC-friendly diagnostics cannot preserve expression information because
	// expressions themselves are not RPC-friendly.
	return nil
}

func (d rpcFriendlyDiag) ExtraInfo() interface{} {
	// RPC-friendly diagnostics always discard any "extra information".
	return nil
}

func init() {
	gob.Register((*rpcFriendlyDiag)(nil))
}
