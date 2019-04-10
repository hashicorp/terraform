package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type MatchAllQuery struct {
}

func (q *MatchAllQuery) Type() QueryType {
	return QueryType_MatchAllQuery
}

func (q *MatchAllQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.MatchAllQuery{}
	data, err := proto.Marshal(query)
	return data, err
}

func (q *MatchAllQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
