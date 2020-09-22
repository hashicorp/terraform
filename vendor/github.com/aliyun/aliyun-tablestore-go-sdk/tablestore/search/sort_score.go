package search

import "github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"

type ScoreSort struct {
	Order *SortOrder
}

func NewScoreSort() *ScoreSort {
	return &ScoreSort{
		Order: SortOrder_DESC.Enum(),
	}
}

func (s *ScoreSort) ProtoBuffer() (*otsprotocol.Sorter, error) {
	pbScoreSort := &otsprotocol.ScoreSort{}
	if s.Order != nil {
		pbOrder, err := s.Order.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		pbScoreSort.Order = pbOrder
	}
	pbSorter := &otsprotocol.Sorter{
		ScoreSort: pbScoreSort,
	}
	return pbSorter, nil
}
