// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"sync"

	"github.com/zclconf/go-cty/cty"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/schemarepo"
)

// policyRunOpts describes the relationship run of a plan or apply walk. A walk
// has a run only when the policy plugin announced the relationships
// capability.
type policyRunOpts struct {
	Stage    proto.EvaluationStage
	PlanMode proto.PlanMode
	Targeted bool
	Schemas  *schemarepo.Schemas

	// AppliedChanges is set for apply runs only: the applied plan's resource
	// instance changes, copied before the walk because the apply walk removes
	// changes from the working changes as it applies them.
	AppliedChanges []*plans.ResourceInstanceChange
}

// policyProviderTable records the provider configurations configured during
// a walk, so the policy plugin can tell whether two resource instances are
// managed through equivalent provider configurations without seeing the
// configurations themselves.
type policyProviderTable struct {
	mu     sync.Mutex
	key    [32]byte
	byAddr map[string]*proto.ProviderInstance
	order  []*proto.ProviderInstance
}

func newPolicyProviderTable() *policyProviderTable {
	t := &policyProviderTable{
		byAddr: make(map[string]*proto.ProviderInstance),
	}
	if _, err := rand.Read(t.key[:]); err != nil {
		// crypto/rand doesn't fail on supported platforms.
		panic(err)
	}
	return t
}

// configured records the configuration of the given provider configuration.
// cfgType is the implied type of the provider's configuration schema, or
// cty.NilType if the schema isn't available.
func (t *policyProviderTable) configured(addr addrs.AbsProviderConfig, cfg cty.Value, cfgType cty.Type) {
	class := t.class(addr.Provider, cfg, cfgType)

	t.mu.Lock()
	defer t.mu.Unlock()
	entry := t.entryLocked(addr)
	entry.Known = class != nil
	entry.ConfigClass = class
}

func (t *policyProviderTable) class(provider addrs.Provider, cfg cty.Value, cfgType cty.Type) []byte {
	if cfgType == cty.NilType || cfg == cty.NilVal {
		return nil
	}
	cfg, _ = cfg.UnmarkDeep()
	if !cfg.IsWhollyKnown() {
		return nil
	}
	encoded, err := ctymsgpack.Marshal(cfg, cfgType)
	if err != nil {
		log.Printf("[WARN] policy: failed to encode the configuration of %s for relationship checks: %s", provider, err)
		return nil
	}
	mac := hmac.New(sha256.New, t.key[:])
	mac.Write([]byte(provider.String()))
	mac.Write([]byte{0})
	mac.Write(encoded)
	return mac.Sum(nil)
}

// idFor returns the id of the given provider configuration, adding an entry
// that isn't known if the provider configuration wasn't configured.
func (t *policyProviderTable) idFor(addr addrs.AbsProviderConfig) uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.entryLocked(addr).Id
}

func (t *policyProviderTable) entryLocked(addr addrs.AbsProviderConfig) *proto.ProviderInstance {
	key := addr.String()
	if entry, ok := t.byAddr[key]; ok {
		return entry
	}
	entry := &proto.ProviderInstance{
		Id:            uint32(len(t.order) + 1),
		ConfigAddress: key,
		Source:        addr.Provider.String(),
	}
	t.byAddr[key] = entry
	t.order = append(t.order, entry)
	return entry
}

// all returns copies of all entries in order of their ids.
func (t *policyProviderTable) all() []*proto.ProviderInstance {
	t.mu.Lock()
	defer t.mu.Unlock()
	ret := make([]*proto.ProviderInstance, len(t.order))
	for i, entry := range t.order {
		ret[i] = &proto.ProviderInstance{
			Id:            entry.Id,
			ConfigAddress: entry.ConfigAddress,
			Source:        entry.Source,
			ConfigClass:   append([]byte(nil), entry.ConfigClass...),
			Known:         entry.Known,
		}
	}
	return ret
}
