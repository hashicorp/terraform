package stackplan

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func LoadFromProto(msgs []*anypb.Any) (*Plan, error) {
	ret := &Plan{}

	foundHeader := false
	for i, rawMsg := range msgs {
		msg, err := anypb.UnmarshalNew(rawMsg, proto.UnmarshalOptions{
			// Just the default unmarshalling options
		})
		if err != nil {
			return nil, fmt.Errorf("invalid raw message %d: %w", i, err)
		}

		// The references to specific message types below ensure that
		// the protobuf descriptors for these types are included in the
		// compiled program, and thus available in the global protobuf
		// registry that anypb.UnmarshalNew relies on above.
		switch msg := msg.(type) {

		case *tfstackdata1.PlanHeader:
			wantVersion := version.String()
			gotVersion := msg.TerraformVersion
			if gotVersion != wantVersion {
				return nil, fmt.Errorf("plan was created by Terraform %s, but this is Terraform %s", gotVersion, wantVersion)
			}
			foundHeader = true

		case *tfstackdata1.PlanApplyable:
			ret.Applyable = msg.Applyable

		case *tfstackdata1.PlanComponentInstance:
			addr, diags := stackaddrs.ParseAbsComponentInstanceStr(msg.ComponentInstanceAddr)
			if diags.HasErrors() {
				// Should not get here because the address we're parsing
				// should've been produced by this same version of Terraform.
				return nil, fmt.Errorf("invalid component instance address syntax in %q", msg.ComponentInstanceAddr)
			}
			if !ret.Components.HasKey(addr) {
				ret.Components.Put(addr, &Component{
					ResourceInstanceChangedOutside: addrs.MakeMap[addrs.AbsResourceInstance, *plans.ResourceInstanceChangeSrc](),
					ResourceInstancePlanned:        addrs.MakeMap[addrs.AbsResourceInstance, *plans.ResourceInstanceChangeSrc](),
				})
			}
			c := ret.Components.Get(addr)
			err := c.PlanTimestamp.UnmarshalText([]byte(msg.PlanTimestamp))
			if err != nil {
				return nil, fmt.Errorf("invalid plan timestamp %q for %s", msg.PlanTimestamp, addr)
			}

		case *tfstackdata1.PlanResourceInstanceChangePlanned:
			if msg.Change == nil {
				return nil, fmt.Errorf("%T has nil Change", msg)
			}
			cAddr, diags := stackaddrs.ParseAbsComponentInstanceStr(msg.ComponentInstanceAddr)
			if diags.HasErrors() {
				return nil, fmt.Errorf("invalid component instance address syntax in %q", msg.ComponentInstanceAddr)
			}
			c, ok := ret.Components.GetOk(cAddr)
			if !ok {
				return nil, fmt.Errorf("resource instance change for unannounced component instance %s", cAddr)
			}
			riPlan, err := planfile.ResourceChangeFromProto(msg.Change)
			if err != nil {
				return nil, fmt.Errorf("invalid resource instance change: %w", err)
			}
			c.ResourceInstancePlanned.Put(riPlan.Addr, riPlan)

		case *tfstackdata1.PlanResourceInstanceChangeOutside:
			if msg.Change == nil {
				return nil, fmt.Errorf("%T has nil Change", msg)
			}
			cAddr, diags := stackaddrs.ParseAbsComponentInstanceStr(msg.ComponentInstanceAddr)
			if diags.HasErrors() {
				return nil, fmt.Errorf("invalid component instance address syntax in %q", msg.ComponentInstanceAddr)
			}
			c, ok := ret.Components.GetOk(cAddr)
			if !ok {
				return nil, fmt.Errorf("resource instance change for unannounced component instance %s", cAddr)
			}
			riPlan, err := planfile.ResourceChangeFromProto(msg.Change)
			if err != nil {
				return nil, fmt.Errorf("invalid resource instance change: %w", err)
			}
			c.ResourceInstanceChangedOutside.Put(riPlan.Addr, riPlan)

		default:
			// Should not get here, because a stack plan can only be loaded by
			// the same version of Terraform that created it, and the above
			// should cover everything this version of Terraform can possibly
			// emit during PlanStackChanges.
			return nil, fmt.Errorf("unsupported raw message type %T at index %d", msg, i)
		}
	}

	// If we got through all of the messages without encountering at least
	// one *PlanHeader then we'll abort because we may have lost part of the
	// plan sequence somehow.
	if !foundHeader {
		return nil, fmt.Errorf("missing PlanHeader")
	}

	return ret, nil
}
