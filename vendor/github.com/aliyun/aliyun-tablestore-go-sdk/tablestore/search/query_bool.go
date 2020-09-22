package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type BoolQuery struct {
	MustQueries        []Query
	MustNotQueries     []Query
	FilterQueries      []Query
	ShouldQueries      []Query
	MinimumShouldMatch *int32
}

func (q *BoolQuery) Type() QueryType {
	return QueryType_BoolQuery
}

func (q *BoolQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.BoolQuery{}
	if q.MustQueries != nil {
		pbMustQs := make([]*otsprotocol.Query, 0)
		for _, mustQ := range q.MustQueries {
			pbQ, err := mustQ.ProtoBuffer()
			if err != nil {
				return nil, err
			}
			pbMustQs = append(pbMustQs, pbQ)
		}
		query.MustQueries = pbMustQs
	}
	if q.MustNotQueries != nil {
		pbMustNotQs := make([]*otsprotocol.Query, 0)
		for _, mustNotQ := range q.MustNotQueries {
			pbQ, err := mustNotQ.ProtoBuffer()
			if err != nil {
				return nil, err
			}
			pbMustNotQs = append(pbMustNotQs, pbQ)
		}
		query.MustNotQueries = pbMustNotQs
	}
	if q.FilterQueries != nil {
		pbFilterQs := make([]*otsprotocol.Query, 0)
		for _, filterQ := range q.FilterQueries {
			pbQ, err := filterQ.ProtoBuffer()
			if err != nil {
				return nil, err
			}
			pbFilterQs = append(pbFilterQs, pbQ)
		}
		query.FilterQueries = pbFilterQs
	}
	if q.ShouldQueries != nil {
		pbShouldQs := make([]*otsprotocol.Query, 0)
		for _, shouldQ := range q.ShouldQueries {
			pbQ, err := shouldQ.ProtoBuffer()
			if err != nil {
				return nil, err
			}
			pbShouldQs = append(pbShouldQs, pbQ)
		}
		query.ShouldQueries = pbShouldQs
	}
	if (q.MinimumShouldMatch != nil) {
		query.MinimumShouldMatch = q.MinimumShouldMatch
	}
	data, err := proto.Marshal(query)
	return data, err
}

func (q *BoolQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
