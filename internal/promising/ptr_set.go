// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package promising

type ptrSet[T any] map[*T]struct{}

func (s ptrSet[T]) Add(p *T) {
	s[p] = struct{}{}
}

func (s ptrSet[T]) Remove(p *T) {
	delete(s, p)
}

func (s ptrSet[T]) Has(p *T) bool {
	_, ret := s[p]
	return ret
}

type promiseSet = ptrSet[promise]
