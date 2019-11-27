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

package codec

import "github.com/tikv/client-go/key"

var (
	tablePrefix     = []byte{'t'}
	recordPrefixSep = []byte("_r")
	indexPrefixSep  = []byte("_i")
)

const (
	idLen                 = 8
	prefixLen             = 1 + idLen /*tableID*/ + 2
	recordRowKeyLen       = prefixLen + idLen /*handle*/
	tablePrefixLength     = 1
	recordPrefixSepLength = 2
)

// appendTableRecordPrefix appends table record prefix  "t[tableID]_r".
func appendTableRecordPrefix(buf []byte, tableID int64) []byte {
	buf = append(buf, tablePrefix...)
	buf = EncodeInt(buf, tableID)
	buf = append(buf, recordPrefixSep...)
	return buf
}

// GenTableRecordPrefix composes record prefix with tableID: "t[tableID]_r".
func GenTableRecordPrefix(tableID int64) key.Key {
	buf := make([]byte, 0, len(tablePrefix)+8+len(recordPrefixSep))
	return appendTableRecordPrefix(buf, tableID)
}

// appendTableIndexPrefix appends table index prefix  "t[tableID]_i".
func appendTableIndexPrefix(buf []byte, tableID int64) []byte {
	buf = append(buf, tablePrefix...)
	buf = EncodeInt(buf, tableID)
	buf = append(buf, indexPrefixSep...)
	return buf
}

// GenTableIndexPrefix composes index prefix with tableID: "t[tableID]_i".
func GenTableIndexPrefix(tableID int64) key.Key {
	buf := make([]byte, 0, len(tablePrefix)+8+len(indexPrefixSep))
	return appendTableIndexPrefix(buf, tableID)
}

// EncodeTableIndexPrefix encodes index prefix with tableID and idxID.
func EncodeTableIndexPrefix(tableID, idxID int64) key.Key {
	key := make([]byte, 0, prefixLen)
	key = appendTableIndexPrefix(key, tableID)
	key = EncodeInt(key, idxID)
	return key
}
