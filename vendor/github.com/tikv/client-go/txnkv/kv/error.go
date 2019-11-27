// Copyright 2019 PingCAP, Inc.
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
	"github.com/pkg/errors"
)

var (
	// ErrNotExist is used when try to get an entry with an unexist key from KV store.
	ErrNotExist = errors.New("key not exist")
	// ErrCannotSetNilValue is the error when sets an empty value.
	ErrCannotSetNilValue = errors.New("can not set nil value")
	// ErrTxnTooLarge is the error when transaction is too large, lock time reached the maximum value.
	ErrTxnTooLarge = errors.New("transaction is too large")
	// ErrEntryTooLarge is the error when a key value entry is too large.
	ErrEntryTooLarge = errors.New("entry is too large")
	// ErrKeyExists returns when key is already exist.
	ErrKeyExists = errors.New("key already exist")
	// ErrInvalidTxn is the error that using a transaction after calling Commit or Rollback.
	ErrInvalidTxn = errors.New("invalid transaction")
)

// IsErrNotFound checks if err is a kind of NotFound error.
func IsErrNotFound(err error) bool {
	return errors.Cause(err) == ErrNotExist
}
