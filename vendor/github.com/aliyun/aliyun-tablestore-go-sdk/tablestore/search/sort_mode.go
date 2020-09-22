package search

import (
	"errors"
	"fmt"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
)

type SortMode int8

const (
	SortMode_Min SortMode = 0
	SortMode_Max SortMode = 1
	SortMode_Avg SortMode = 2
)

func (x SortMode) Enum() *SortMode {
	p := new(SortMode)
	*p = x
	return p
}

func (m *SortMode) ProtoBuffer() (*otsprotocol.SortMode, error) {
	if m == nil {
		return nil, errors.New("sort mode is nil")
	}
	if *m == SortMode_Min {
		return otsprotocol.SortMode_SORT_MODE_MIN.Enum(), nil
	} else if *m == SortMode_Max {
		return otsprotocol.SortMode_SORT_MODE_MAX.Enum(), nil
	} else if *m == SortMode_Avg {
		return otsprotocol.SortMode_SORT_MODE_AVG.Enum(), nil
	} else {
		return nil, errors.New("unknown sort mode: " + fmt.Sprintf("%#v", *m))
	}
}
