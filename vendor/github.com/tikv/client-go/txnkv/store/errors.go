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

package store

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/tikv/client-go/key"
)

// TxnRetryableMark is used to direct user to restart a transaction.
// TiDB decides whether to retry transaction by checking if error message contains
// string "try again later" literally. The common usage is `errors.Annotate(err, TxnRetryableMark)`.
// Note that it should be only used if i) the error occurs inside a transaction
// and ii) the error is not totally unexpected and hopefully will recover soon.
const TxnRetryableMark = "[try again later]"

var (
	// ErrResultUndetermined means that the commit status is unknown.
	ErrResultUndetermined = errors.New("result undetermined")
	// ErrNotImplemented returns when a function is not implemented yet.
	ErrNotImplemented = errors.New("not implemented")
	// ErrPDServerTimeout is the error that PD does not repond in time.
	ErrPDServerTimeout = errors.New("PD server timeout")
	// ErrStartTSFallBehind is the error a transaction runs too long and data
	// loaded from TiKV may out of date because of GC.
	ErrStartTSFallBehind = errors.New("StartTS may fall behind safePoint")
)

// ErrKeyAlreadyExist is the error that a key exists in TiKV when it should not.
type ErrKeyAlreadyExist key.Key

func (e ErrKeyAlreadyExist) Error() string {
	return fmt.Sprintf("key already exists: %q", e)
}
