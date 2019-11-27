// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package kv

import (
	"context"

	"github.com/tikv/client-go/key"
)

type mockSnapshot struct {
	store MemBuffer
}

func (s *mockSnapshot) Get(ctx context.Context, k key.Key) ([]byte, error) {
	return s.store.Get(ctx, k)
}

func (s *mockSnapshot) SetPriority(priority int) {

}

func (s *mockSnapshot) BatchGet(ctx context.Context, keys []key.Key) (map[string][]byte, error) {
	m := make(map[string][]byte)
	for _, k := range keys {
		v, err := s.store.Get(ctx, k)
		if IsErrNotFound(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		m[string(k)] = v
	}
	return m, nil
}

func (s *mockSnapshot) Iter(ctx context.Context, k key.Key, upperBound key.Key) (Iterator, error) {
	return s.store.Iter(ctx, k, upperBound)
}

func (s *mockSnapshot) IterReverse(ctx context.Context, k key.Key) (Iterator, error) {
	return s.store.IterReverse(ctx, k)
}
