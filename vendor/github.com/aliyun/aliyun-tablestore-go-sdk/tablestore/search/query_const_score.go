package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type ConstScoreQuery struct {
	Filter Query
}

func (q *ConstScoreQuery) Type() QueryType {
	return QueryType_ConstScoreQuery
}

func (q *ConstScoreQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.ConstScoreQuery{}
	pbQ, err := q.Filter.ProtoBuffer()
	if err != nil {
		return nil, err
	}
	query.Filter = pbQ
	data, err := proto.Marshal(query)
	return data, err
}

func (q *ConstScoreQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
