package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type SearchQuery interface {
	Serialize() ([]byte, error)
}

type searchQuery struct {
	Offset        int32
	Limit         int32
	Query         Query
	Collapse      *Collapse
	Sort          *Sort
	GetTotalCount bool
	Token         []byte
}

func NewSearchQuery() *searchQuery {
	return &searchQuery{
		Offset:        -1,
		Limit:         -1,
		GetTotalCount: false,
	}
}

func (s *searchQuery) SetOffset(offset int32) *searchQuery {
	s.Offset = offset
	return s
}

func (s *searchQuery) SetLimit(limit int32) *searchQuery {
	s.Limit = limit
	return s
}

func (s *searchQuery) SetQuery(query Query) *searchQuery {
	s.Query = query
	return s
}

func (s *searchQuery) SetCollapse(collapse *Collapse) *searchQuery {
	s.Collapse = collapse
	return s
}

func (s *searchQuery) SetSort(sort *Sort) *searchQuery {
	s.Sort = sort
	return s
}

func (s *searchQuery) SetGetTotalCount(getTotalCount bool) *searchQuery {
	s.GetTotalCount = getTotalCount
	return s
}

func (s *searchQuery) SetToken(token []byte) *searchQuery {
	s.Token = token
	s.Sort = nil
	return s
}

func (s *searchQuery) Serialize() ([]byte, error) {
	search_query := &otsprotocol.SearchQuery{}
	if s.Offset >= 0 {
		search_query.Offset = &s.Offset
	}
	if s.Limit >= 0 {
		search_query.Limit = &s.Limit
	}
	if s.Query != nil {
		pbQuery, err := s.Query.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		search_query.Query = pbQuery
	}
	if s.Collapse != nil {
		pbCollapse, err := s.Collapse.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		search_query.Collapse = pbCollapse
	}
	if s.Sort != nil {
		pbSort, err := s.Sort.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		search_query.Sort = pbSort
	}
	search_query.GetTotalCount = &s.GetTotalCount
	if s.Token != nil && len(s.Token) > 0 {
		search_query.Token = s.Token
	}
	data, err := proto.Marshal(search_query)
	return data, err
}
