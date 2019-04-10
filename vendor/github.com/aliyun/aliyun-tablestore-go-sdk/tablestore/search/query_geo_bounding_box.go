package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type GeoBoundingBoxQuery struct {
	FieldName   string
	TopLeft     string
	BottomRight string
}

func (q *GeoBoundingBoxQuery) Type() QueryType {
	return QueryType_GeoBoundingBoxQuery
}

func (q *GeoBoundingBoxQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.GeoBoundingBoxQuery{}
	query.FieldName = &q.FieldName
	query.TopLeft = &q.TopLeft
	query.BottomRight = &q.BottomRight
	data, err := proto.Marshal(query)
	return data, err
}

func (q *GeoBoundingBoxQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
