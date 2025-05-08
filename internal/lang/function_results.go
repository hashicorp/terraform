// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

type priorResultHash struct {
	hash [sha256.Size]byte
	// when the result was from a current run, we keep a record of the result
	// value to aid in debugging. Results stored in the plan will only have the
	// hash to avoid bloating the plan with what could be many very large
	// values.
	value cty.Value
}

type FunctionResults struct {
	mu sync.Mutex
	// results stores the prior result from a function call, keyed by
	// the hash of the function name and arguments.
	results map[[sha256.Size]byte]priorResultHash
}

// NewFunctionResultsTable initializes a mapping of function calls to prior
// results used to validate function calls. The hashes argument is an
// optional slice of prior result hashes used to preload the cache.
func NewFunctionResultsTable(hashes []FunctionResultHash) *FunctionResults {
	res := &FunctionResults{
		results: make(map[[sha256.Size]byte]priorResultHash),
	}

	res.insertHashes(hashes)
	return res
}

// CheckPrior compares the function call against any cached results, and returns
// an error if the result does not match a prior call.
func (f *FunctionResults) CheckPrior(name string, args []cty.Value, result cty.Value) error {
	return f.CheckPriorProvider(addrs.Provider{}, name, args, result)
}

// CheckPriorProvider compares the provider function call against any cached
// results, and returns an error if the result does not match a prior call.
func (f *FunctionResults) CheckPriorProvider(provider addrs.Provider, name string, args []cty.Value, result cty.Value) error {
	argSum := sha256.New()

	if !provider.IsZero() {
		io.WriteString(argSum, provider.String()+"|")
	}
	io.WriteString(argSum, name)

	for _, arg := range args {
		// cty.Values have a Hash method, but it is not collision resistant. We
		// are going to rely on the GoString formatting instead, which gives
		// detailed results for all values.
		io.WriteString(argSum, "|"+arg.GoString())
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	argHash := [sha256.Size]byte(argSum.Sum(nil))
	resHash := sha256.Sum256([]byte(result.GoString()))

	res, ok := f.results[argHash]
	if !ok {
		f.results[argHash] = priorResultHash{
			hash:  resHash,
			value: result,
		}
		return nil
	}

	if resHash != res.hash {
		provPrefix := ""
		if !provider.IsZero() {
			provPrefix = fmt.Sprintf("provider %s ", provider)
		}
		// Log the args for debugging in case the hcl context is
		// insufficient. The error should be adequate most of the time, and
		// could already be quite long, so we don't want to add all
		// arguments too.
		log.Printf("[ERROR] %sfunction %s returned an inconsistent result with args: %#v\n", provPrefix, name, args)
		// The hcl package will add the necessary context around the error in
		// the diagnostic, but we add the differing results when we can.
		if res.value != cty.NilVal {
			return fmt.Errorf("function returned an inconsistent result,\nwas: %#v,\nnow: %#v", res.value, result)
		}
		return fmt.Errorf("function returned an inconsistent result")
	}

	return nil
}

// insertHashes insert key-value pairs to the functionResults map. This is used
// to preload stored values before any Verify calls are made.
func (f *FunctionResults) insertHashes(hashes []FunctionResultHash) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, res := range hashes {
		f.results[[sha256.Size]byte(res.Key)] = priorResultHash{
			hash: [sha256.Size]byte(res.Result),
		}
	}
}

// FunctionResultHash contains the key and result hash values from a prior function
// call.
type FunctionResultHash struct {
	Key    []byte
	Result []byte
}

// copy the hash values into a struct which can be recorded in the plan.
func (f *FunctionResults) GetHashes() []FunctionResultHash {
	f.mu.Lock()
	defer f.mu.Unlock()

	var res []FunctionResultHash
	for k, r := range f.results {
		res = append(res, FunctionResultHash{Key: k[:], Result: r.hash[:]})
	}
	return res
}
