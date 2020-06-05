package search

import (
	"errors"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type RangeQuery struct {
	FieldName    string
	From         interface{}
	To           interface{}
	IncludeLower bool
	IncludeUpper bool
}

func (q *RangeQuery) GT(value interface{}) {
	q.from(value, false)
}

func (q *RangeQuery) GTE(value interface{}) {
	q.from(value, true)
}

func (q *RangeQuery) LT(value interface{}) {
	q.to(value, false)
}

func (q *RangeQuery) LTE(value interface{}) {
	q.to(value, true)
}

func (q *RangeQuery) from(value interface{}, includeLower bool) {
	q.From = value
	q.IncludeLower = includeLower
}

func (q *RangeQuery) to(value interface{}, includeUpper bool) {
	q.To = value
	q.IncludeUpper = includeUpper
}

func (q *RangeQuery) Type() QueryType {
	return QueryType_RangeQuery
}

func (q *RangeQuery) Serialize() ([]byte, error) {
	if q.FieldName == "" {
		return nil, errors.New("RangeQuery: fieldName not set.")
	}
	query := &otsprotocol.RangeQuery{}
	query.FieldName = &q.FieldName
	if q.From != nil {
		vFrom, err := ToVariantValue(q.From)
		if err != nil {
			return nil, err
		}
		query.RangeFrom = ([]byte)(vFrom)
	}
	if q.To != nil {
		vTo, err := ToVariantValue(q.To)
		if err != nil {
			return nil, err
		}
		query.RangeTo = ([]byte)(vTo)
	}
	query.IncludeLower = &q.IncludeLower
	query.IncludeUpper = &q.IncludeUpper
	data, err := proto.Marshal(query)
	return data, err
}

func (q *RangeQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
