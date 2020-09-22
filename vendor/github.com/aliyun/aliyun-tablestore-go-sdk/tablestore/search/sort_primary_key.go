package search

import "github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"

type PrimaryKeySort struct {
	Order *SortOrder
}

func NewPrimaryKeySort() *PrimaryKeySort {
	return &PrimaryKeySort{
		Order: SortOrder_ASC.Enum(),
	}
}

func (s *PrimaryKeySort) ProtoBuffer() (*otsprotocol.Sorter, error) {
	pbPrimaryKeySort := &otsprotocol.PrimaryKeySort{}
	if s.Order != nil {
		pbOrder, err := s.Order.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		pbPrimaryKeySort.Order = pbOrder
	}
	pbSorter := &otsprotocol.Sorter{
		PkSort: pbPrimaryKeySort,
	}
	return pbSorter, nil
}
