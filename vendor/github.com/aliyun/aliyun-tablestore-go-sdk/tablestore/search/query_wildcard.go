package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type WildcardQuery struct {
	FieldName string
	Value     string
}

func (q *WildcardQuery) Type() QueryType {
	return QueryType_WildcardQuery
}

func (q *WildcardQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.WildcardQuery{}
	query.FieldName = &q.FieldName
	query.Value = &q.Value
	data, err := proto.Marshal(query)
	return data, err
}

func (q *WildcardQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
