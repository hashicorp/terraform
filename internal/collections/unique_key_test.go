// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package collections

type testingKey string

// testingKey is its own UniqueKey, because it's already a comparable type
var _ UniqueKey[testingKey] = testingKey("")
var _ UniqueKeyer[testingKey] = testingKey("")

func (k testingKey) IsUniqueKey(testingKey) {}

// UniqueKey implements UniqueKeyer.
func (k testingKey) UniqueKey() UniqueKey[testingKey] {
	return UniqueKey[testingKey](k)
}
