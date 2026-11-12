// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package schema

import (
	"sync"
)

// MapFieldWriter writes data into a single map[string]string structure.
type MapFieldWriter struct {
	Schema map[string]*Schema

	lock   sync.Mutex
	result map[string]string
}

// Map returns the underlying map that is being written to.
func (w *MapFieldWriter) Map() map[string]string {
	w.lock.Lock()
	defer w.lock.Unlock()
	if w.result == nil {
		w.result = make(map[string]string)
	}

	return w.result
}
