package search

import "github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"

type QueryType int

const (
	QueryType_None                QueryType = 0
	QueryType_MatchQuery          QueryType = 1
	QueryType_MatchPhraseQuery    QueryType = 2
	QueryType_TermQuery           QueryType = 3
	QueryType_RangeQuery          QueryType = 4
	QueryType_PrefixQuery         QueryType = 5
	QueryType_BoolQuery           QueryType = 6
	QueryType_ConstScoreQuery     QueryType = 7
	QueryType_FunctionScoreQuery  QueryType = 8
	QueryType_NestedQuery         QueryType = 9
	QueryType_WildcardQuery       QueryType = 10
	QueryType_MatchAllQuery       QueryType = 11
	QueryType_GeoBoundingBoxQuery QueryType = 12
	QueryType_GeoDistanceQuery    QueryType = 13
	QueryType_GeoPolygonQuery     QueryType = 14
	QueryType_TermsQuery          QueryType = 15
)

func (q QueryType) Enum() *QueryType {
	newQuery := q
	return &newQuery
}

func (q QueryType) ToPB() *otsprotocol.QueryType {
	switch q {
	case QueryType_None:
		return nil
	case QueryType_MatchQuery:
		return otsprotocol.QueryType_MATCH_QUERY.Enum()
	case QueryType_MatchPhraseQuery:
		return otsprotocol.QueryType_MATCH_PHRASE_QUERY.Enum()
	case QueryType_TermQuery:
		return otsprotocol.QueryType_TERM_QUERY.Enum()
	case QueryType_RangeQuery:
		return otsprotocol.QueryType_RANGE_QUERY.Enum()
	case QueryType_PrefixQuery:
		return otsprotocol.QueryType_PREFIX_QUERY.Enum()
	case QueryType_BoolQuery:
		return otsprotocol.QueryType_BOOL_QUERY.Enum()
	case QueryType_ConstScoreQuery:
		return otsprotocol.QueryType_CONST_SCORE_QUERY.Enum()
	case QueryType_FunctionScoreQuery:
		return otsprotocol.QueryType_FUNCTION_SCORE_QUERY.Enum()
	case QueryType_NestedQuery:
		return otsprotocol.QueryType_NESTED_QUERY.Enum()
	case QueryType_WildcardQuery:
		return otsprotocol.QueryType_WILDCARD_QUERY.Enum()
	case QueryType_MatchAllQuery:
		return otsprotocol.QueryType_MATCH_ALL_QUERY.Enum()
	case QueryType_GeoBoundingBoxQuery:
		return otsprotocol.QueryType_GEO_BOUNDING_BOX_QUERY.Enum()
	case QueryType_GeoDistanceQuery:
		return otsprotocol.QueryType_GEO_DISTANCE_QUERY.Enum()
	case QueryType_GeoPolygonQuery:
		return otsprotocol.QueryType_GEO_POLYGON_QUERY.Enum()
	case QueryType_TermsQuery:
		return otsprotocol.QueryType_TERMS_QUERY.Enum()
	default:
		panic("unexpected")
	}
}

type Query interface {
	Type() QueryType
	Serialize() ([]byte, error)
	ProtoBuffer() (*otsprotocol.Query, error)
}

func BuildPBForQuery(q Query) (*otsprotocol.Query, error) {
	query := &otsprotocol.Query{}
	query.Type = q.Type().ToPB()
	data, err := q.Serialize()
	if err != nil {
		return nil, err
	}
	query.Query = data
	return query, nil
}
