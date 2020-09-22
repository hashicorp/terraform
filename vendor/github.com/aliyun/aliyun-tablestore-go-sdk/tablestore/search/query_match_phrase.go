package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type MatchPhraseQuery struct {
	FieldName string
	Text      string
}

func (q *MatchPhraseQuery) Type() QueryType {
	return QueryType_MatchPhraseQuery
}

func (q *MatchPhraseQuery) Serialize() ([]byte, error) {
	query := &otsprotocol.MatchPhraseQuery{}
	query.FieldName = &q.FieldName
	query.Text = &q.Text
	data, err := proto.Marshal(query)
	return data, err
}

func (q *MatchPhraseQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
