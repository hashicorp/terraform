package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type GeoDistanceQuery struct {
	FieldName       string
	CenterPoint     string
	DistanceInMeter float64
}

func (q *GeoDistanceQuery) Type() QueryType {
	return QueryType_GeoDistanceQuery
}

func (q *GeoDistanceQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.GeoDistanceQuery{}
	query.FieldName = &q.FieldName
	query.CenterPoint = &q.CenterPoint
	query.Distance = &q.DistanceInMeter
	data, err := proto.Marshal(query)
	return data, err
}

func (q *GeoDistanceQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
