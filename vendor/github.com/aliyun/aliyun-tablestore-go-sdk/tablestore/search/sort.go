package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
)

type Sorter interface {
	ProtoBuffer() (*otsprotocol.Sorter, error)
}

type Sort struct {
	Sorters []Sorter
}

func (s *Sort) ProtoBuffer() (*otsprotocol.Sort, error) {
	pbSort := &otsprotocol.Sort{}
	pbSortors := make([]*otsprotocol.Sorter, 0)
	for _, fs := range s.Sorters {
		pbFs, err := fs.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		pbSortors = append(pbSortors, pbFs)
	}
	pbSort.Sorter = pbSortors
	return pbSort, nil
}
