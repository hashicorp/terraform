package rpcapi

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/rpcapi/tfcore1"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func planOptsFromProto(from *tfcore1.PlanOptions) (*terraform.PlanOpts, error) {
	if from == nil {
		return terraform.DefaultPlanOpts, nil
	}
	ret := &terraform.PlanOpts{}

	var err error
	ret.Mode, err = planModeFromProto(from.Mode)
	if err != nil {
		return nil, err
	}

	ret.SkipRefresh = from.SkipRefresh

	for _, protoAddr := range from.ForceReplace {
		addr, err := absResourceInstanceAddrFromProto(protoAddr)
		if err != nil {
			return nil, err
		}
		ret.ForceReplace = append(ret.ForceReplace, addr)
	}

	if len(from.VariableDefs) > 0 {
		ret.SetVariables = make(terraform.InputValues, len(from.VariableDefs))
		for name, protoVal := range from.VariableDefs {
			val, err := dynamicValueFromProto(protoVal)
			if err != nil {
				return nil, err
			}
			ret.SetVariables[name] = &terraform.InputValue{
				Value:      val,
				SourceType: terraform.ValueFromCaller,
			}
		}
	}

	// NOTE: tfcore1.PlanOptions currently intentionally doesn't implement
	// targeting, because that requires some pretty complex dynamic address
	// handling and isn't a recommended usage pattern anyway.

	return ret, nil
}

func dynamicValueFromProto(from *tfcore1.DynamicValue) (cty.Value, error) {
	if from == nil {
		return cty.NilVal, status.Errorf(codes.InvalidArgument, "missing required dynamic value")
	}

	ty, err := ctyjson.UnmarshalType(from.TypeJson)
	if err != nil {
		return cty.NilVal, status.Errorf(codes.InvalidArgument, "invalid dynamic value type: %s", err)
	}

	v, err := ctymsgpack.Unmarshal(from.ValueMsgpack, ty)
	if err != nil {
		return cty.NilVal, status.Errorf(codes.InvalidArgument, "invalid dynamic value: %s", err)
	}

	if from.Sensitive {
		v = v.Mark(marks.Sensitive)
	}

	return v, nil
}

func dynamicValueToProto(from cty.Value) (*tfcore1.DynamicValue, error) {
	from, fromMarks := from.UnmarkDeep()
	sensitive := false
	for mark := range fromMarks {
		if mark == marks.Sensitive {
			sensitive = true
		} else {
			return nil, status.Errorf(codes.Internal, "unsupported value mark %#v", mark)
		}
	}

	tyJSON, err := ctyjson.MarshalType(from.Type())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encode dynamic value type: %s", err)
	}

	vMsgpack, err := ctymsgpack.Marshal(from, from.Type())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encode dynamic value: %s", err)
	}

	return &tfcore1.DynamicValue{
		TypeJson:     tyJSON,
		ValueMsgpack: vMsgpack,
		Sensitive:    sensitive,
	}, nil
}

func planModeFromProto(from tfcore1.PlanOptions_Mode) (plans.Mode, error) {
	switch from {
	case tfcore1.PlanOptions_NORMAL:
		return plans.NormalMode, nil
	case tfcore1.PlanOptions_REFRESH_ONLY:
		return plans.RefreshOnlyMode, nil
	case tfcore1.PlanOptions_DESTROY:
		return plans.DestroyMode, nil
	default:
		return plans.Mode(0), status.Errorf(codes.InvalidArgument, "unsupported plan mode %q", from.String())
	}
}

func absResourceInstanceAddrFromProto(from *tfcore1.AbsResourceInstance) (addrs.AbsResourceInstance, error) {
	if from == nil {
		return addrs.AbsResourceInstance{}, status.Errorf(codes.InvalidArgument, "missing required resource instance address")
	}

	absResourceAddr, err := absResourceAddrFromProto(from.Resource)
	if err != nil {
		return addrs.AbsResourceInstance{}, err
	}
	key, err := instanceKeyFromProto(from.Key)
	if err != nil {
		return addrs.AbsResourceInstance{}, err
	}
	return addrs.AbsResourceInstance{
		Module: absResourceAddr.Module,
		Resource: addrs.ResourceInstance{
			Resource: absResourceAddr.Resource,
			Key:      key,
		},
	}, nil
}

func absResourceAddrFromProto(from *tfcore1.AbsResource) (addrs.AbsResource, error) {
	if from == nil {
		return addrs.AbsResource{}, status.Errorf(codes.InvalidArgument, "missing required resource address")
	}

	mode, err := resourceModeFromProto(from.Mode)
	if err != nil {
		return addrs.AbsResource{}, err
	}

	moduleInst, err := moduleInstanceAddrFromProto(from.Module)
	if err != nil {
		return addrs.AbsResource{}, err
	}

	return addrs.AbsResource{
		Module: moduleInst,
		Resource: addrs.Resource{
			Mode: mode,
			Type: from.Type,
			Name: from.Name,
		},
	}, nil
}

func resourceModeFromProto(from tfcore1.ResourceMode) (addrs.ResourceMode, error) {
	switch from {
	case tfcore1.ResourceMode_MANAGED:
		return addrs.ManagedResourceMode, nil
	case tfcore1.ResourceMode_DATA:
		return addrs.DataResourceMode, nil
	default:
		return addrs.ResourceMode(0), status.Errorf(codes.InvalidArgument, "unsupported resource mode %q", from.String())
	}
}

func moduleInstanceAddrFromProto(from []*tfcore1.ModuleInstanceStep) (addrs.ModuleInstance, error) {
	if len(from) == 0 {
		return addrs.RootModuleInstance, nil
	}
	ret := make(addrs.ModuleInstance, len(from))
	for i, step := range from {
		if step == nil {
			return nil, status.Errorf(codes.InvalidArgument, "missing required step in module instance address")
		}
		key, err := instanceKeyFromProto(step.Key)
		if err != nil {
			return nil, err
		}
		ret[i] = addrs.ModuleInstanceStep{
			Name:        step.Name,
			InstanceKey: key,
		}
	}
	return ret, nil
}

func instanceKeyFromProto(from *tfcore1.InstanceKey) (addrs.InstanceKey, error) {
	if from == nil {
		return addrs.NoKey, nil
	}
	switch from := from.Key.(type) {
	case *tfcore1.InstanceKey_Int:
		return addrs.IntKey(from.Int), nil
	case *tfcore1.InstanceKey_String_:
		return addrs.StringKey(from.String_), nil
	case nil:
		// A degenerate alternative way to say "no key". The canonical way is
		// for the whole InstanceKey message to be nil.
		return addrs.NoKey, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported instance key type %T", from)
	}
}

func diagnosticsToProto(from tfdiags.Diagnostics) []*tfcore1.Diagnostic {
	if len(from) == 0 {
		return nil
	}
	ret := make([]*tfcore1.Diagnostic, len(from))
	for i, diag := range from {
		protoDiag := &tfcore1.Diagnostic{}

		severity := diag.Severity()
		desc := diag.Description()
		source := diag.Source()

		switch severity {
		case tfdiags.Error:
			protoDiag.Severity = tfcore1.Diagnostic_ERROR
		case tfdiags.Warning:
			protoDiag.Severity = tfcore1.Diagnostic_WARNING
		default:
			panic(fmt.Sprintf("unsupported diagnostic severity %s", severity))
		}

		protoDiag.Summary = desc.Summary
		protoDiag.Detail = desc.Detail
		protoDiag.Address = desc.Address

		if source.Subject != nil {
			protoDiag.Subject = sourceRangeToProto(*source.Subject)
		}
		if source.Context != nil {
			protoDiag.Context = sourceRangeToProto(*source.Context)
		}

		ret[i] = protoDiag
	}
	return ret
}

func sourceRangeToProto(from tfdiags.SourceRange) *tfcore1.SourceRange {
	return &tfcore1.SourceRange{
		Filename: from.Filename,
		Start:    sourcePosToProto(from.Start),
		End:      sourcePosToProto(from.End),
	}
}

func sourcePosToProto(from tfdiags.SourcePos) *tfcore1.SourceRange_Pos {
	return &tfcore1.SourceRange_Pos{
		Line:   int64(from.Line),
		Column: int64(from.Column),
		Byte:   int64(from.Byte),
	}
}
