package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type TermQuery struct {
	FieldName string
	Term      interface{}
}

func (q *TermQuery) Type() QueryType {
	return QueryType_TermQuery
}

func (q *TermQuery) Serialize() ([]byte, error) {
	term := &otsprotocol.TermQuery{}
	term.FieldName = &q.FieldName
	vt, err := ToVariantValue(q.Term)
	if err != nil {
		return nil, err
	}
	term.Term = []byte(vt)
	data, err := proto.Marshal(term)
	return data, err
}

func (q *TermQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
