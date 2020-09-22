package search

import (
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

type TermsQuery struct {
	FieldName string
	Terms     []interface{}
}

func (q *TermsQuery) Type() QueryType {
	return QueryType_TermsQuery
}

func (q *TermsQuery) Serialize() ([]byte, error) {
	term := &otsprotocol.TermsQuery{}
	term.FieldName = &q.FieldName
	term.Terms = make([][]byte, 0)

	for _, value := range q.Terms {
		vt, err := ToVariantValue(value)
		if err != nil {
			return nil, err
		}
		term.Terms = append(term.Terms, []byte(vt))
	}
	data, err := proto.Marshal(term)
	return data, err
}

func (q *TermsQuery) ProtoBuffer() (*otsprotocol.Query, error) {
	return BuildPBForQuery(q)
}
