package search

import (
	"errors"
	"fmt"

	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type QueryOperator int8

const (
	QueryOperator_OR  QueryOperator = 0
	QueryOperator_AND QueryOperator = 1
)

func (x QueryOperator) Enum() *QueryOperator {
	p := new(QueryOperator)
	*p = x
	return p
}

func (o *QueryOperator) ProtoBuffer() (*otsprotocol.QueryOperator, error) {
	if o == nil {
		return nil, errors.New("query operator is nil")
	}
	if *o == QueryOperator_OR {
		return otsprotocol.QueryOperator_OR.Enum(), nil
	} else if *o == QueryOperator_AND {
		return otsprotocol.QueryOperator_AND.Enum(), nil
	} else {
		return nil, errors.New("unknown query operator: " + fmt.Sprintf("%#v", *o))
	}
}

type MatchQuery struct {
	FieldName          string
	Text               string
	MinimumShouldMatch *int32
	Operator           *QueryOperator
}

func (q *MatchQuery) Type() QueryType {
	return QueryType_MatchQuery
}

func (q *MatchQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.MatchQuery{}
	query.FieldName = &q.FieldName
	query.Text = &q.Text
	if q.MinimumShouldMatch != nil {
		query.MinimumShouldMatch = q.MinimumShouldMatch
	}
	if q.Operator != nil {
		pbOperator, err := q.Operator.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		query.Operator = pbOperator
	}
	data, err := proto.Marshal(query)
	return data, err
}

func (q *MatchQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
