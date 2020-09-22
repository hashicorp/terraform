package tablestore

import (
	"fmt"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
	//"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
)

// @class TableStoreClient
// The TableStoreClient, which will connect OTS service for authorization, create/list/
// delete tables/table groups, to get/put/delete a row.
// Note: TableStoreClient is thread-safe.
// TableStoreClient的功能包括连接OTS服务进行验证、创建/列出/删除表或表组、插入/获取/
// 删除/更新行数据
type TableStoreClient struct {
	endPoint        string
	instanceName    string
	accessKeyId     string
	accessKeySecret string
	securityToken   string

	httpClient      IHttpClient
	config          *TableStoreConfig
	random          *rand.Rand
}

type ClientOption func(*TableStoreClient)

type TableStoreHttpClient struct {
	httpClient *http.Client
}

// use this to mock http.client for testing
type IHttpClient interface {
	Do(*http.Request) (*http.Response, error)
	New(*http.Client)
}

func (httpClient *TableStoreHttpClient) Do(req *http.Request) (*http.Response, error) {
	return httpClient.httpClient.Do(req)
}

func (httpClient *TableStoreHttpClient) New(client *http.Client) {
	httpClient.httpClient = client
}

type HTTPTimeout struct {
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration
}

type TableStoreConfig struct {
	RetryTimes         uint
	MaxRetryTime       time.Duration
	HTTPTimeout        HTTPTimeout
	MaxIdleConnections int
}

func NewDefaultTableStoreConfig() *TableStoreConfig {
	httpTimeout := &HTTPTimeout{
		ConnectionTimeout: time.Second * 15,
		RequestTimeout:    time.Second * 30}
	config := &TableStoreConfig{
		RetryTimes:         10,
		HTTPTimeout:        *httpTimeout,
		MaxRetryTime:       time.Second * 5,
		MaxIdleConnections: 2000}
	return config
}

type CreateTableRequest struct {
	TableMeta          *TableMeta
	TableOption        *TableOption
	ReservedThroughput *ReservedThroughput
	StreamSpec         *StreamSpecification
	IndexMetas         []*IndexMeta
}

type CreateIndexRequest struct {
	MainTableName   string
	IndexMeta       *IndexMeta
	IncludeBaseData bool
}

type DeleteIndexRequest struct {
	MainTableName string
	IndexName     string
}

type ResponseInfo struct {
	RequestId string
}

type CreateTableResponse struct {
	ResponseInfo
}

type CreateIndexResponse struct {
	ResponseInfo
}

type DeleteIndexResponse struct {
	ResponseInfo
}

type DeleteTableResponse struct {
	ResponseInfo
}

type TableMeta struct {
	TableName      string
	SchemaEntry    []*PrimaryKeySchema
	DefinedColumns []*DefinedColumnSchema
}

type PrimaryKeySchema struct {
	Name   *string
	Type   *PrimaryKeyType
	Option *PrimaryKeyOption
}

type PrimaryKey struct {
	PrimaryKeys []*PrimaryKeyColumn
}

type TableOption struct {
	TimeToAlive, MaxVersion int
}

type ReservedThroughput struct {
	Readcap, Writecap int
}

type ListTableResponse struct {
	TableNames []string
	ResponseInfo
}

type DeleteTableRequest struct {
	TableName string
}

type DescribeTableRequest struct {
	TableName string
}

type DescribeTableResponse struct {
	TableMeta          *TableMeta
	TableOption        *TableOption
	ReservedThroughput *ReservedThroughput
	StreamDetails      *StreamDetails
	IndexMetas         []*IndexMeta
	ResponseInfo
}

type UpdateTableRequest struct {
	TableName          string
	TableOption        *TableOption
	ReservedThroughput *ReservedThroughput
	StreamSpec         *StreamSpecification
}

type UpdateTableResponse struct {
	TableOption        *TableOption
	ReservedThroughput *ReservedThroughput
	StreamDetails      *StreamDetails
	ResponseInfo
}

type ConsumedCapacityUnit struct {
	Read  int32
	Write int32
}

type PutRowResponse struct {
	ConsumedCapacityUnit *ConsumedCapacityUnit
	PrimaryKey           PrimaryKey
	ResponseInfo
}

type DeleteRowResponse struct {
	ConsumedCapacityUnit *ConsumedCapacityUnit
	ResponseInfo
}

type UpdateRowResponse struct {
	Columns              []*AttributeColumn
	ConsumedCapacityUnit *ConsumedCapacityUnit
	ResponseInfo
}

type PrimaryKeyType int32

const (
	PrimaryKeyType_INTEGER PrimaryKeyType = 1
	PrimaryKeyType_STRING PrimaryKeyType = 2
	PrimaryKeyType_BINARY PrimaryKeyType = 3
)

const (
	DefaultRetryInterval = 10
	MaxRetryInterval = 320
)

type PrimaryKeyOption int32

const (
	NONE PrimaryKeyOption = 0
	AUTO_INCREMENT PrimaryKeyOption = 1
	MIN PrimaryKeyOption = 2
	MAX PrimaryKeyOption = 3
)

type PrimaryKeyColumn struct {
	ColumnName       string
	Value            interface{}
	PrimaryKeyOption PrimaryKeyOption
}

func (this *PrimaryKeyColumn) String() string {
	xs := make([]string, 0)
	xs = append(xs, fmt.Sprintf("\"Name\": \"%s\"", this.ColumnName))
	switch this.PrimaryKeyOption {
	case NONE:
		xs = append(xs, fmt.Sprintf("\"Value\": \"%s\"", this.Value))
	case MIN:
		xs = append(xs, "\"Value\": -inf")
	case MAX:
		xs = append(xs, "\"Value\": +inf")
	case AUTO_INCREMENT:
		xs = append(xs, "\"Value\": auto-incr")
	}
	return fmt.Sprintf("{%s}", strings.Join(xs, ", "))
}

type AttributeColumn struct {
	ColumnName string
	Value      interface{}
	Timestamp  int64
}

type TimeRange struct {
	Start    int64
	End      int64
	Specific int64
}

type ColumnToUpdate struct {
	ColumnName   string
	Type         byte
	Timestamp    int64
	HasType      bool
	HasTimestamp bool
	IgnoreValue  bool
	Value        interface{}
}

type RowExistenceExpectation int

const (
	RowExistenceExpectation_IGNORE RowExistenceExpectation = 0
	RowExistenceExpectation_EXPECT_EXIST RowExistenceExpectation = 1
	RowExistenceExpectation_EXPECT_NOT_EXIST RowExistenceExpectation = 2
)

type ComparatorType int32

const (
	CT_EQUAL ComparatorType = 1
	CT_NOT_EQUAL ComparatorType = 2
	CT_GREATER_THAN ComparatorType = 3
	CT_GREATER_EQUAL ComparatorType = 4
	CT_LESS_THAN ComparatorType = 5
	CT_LESS_EQUAL ComparatorType = 6
)

type LogicalOperator int32

const (
	LO_NOT LogicalOperator = 1
	LO_AND LogicalOperator = 2
	LO_OR LogicalOperator = 3
)

type FilterType int32

const (
	FT_SINGLE_COLUMN_VALUE FilterType = 1
	FT_COMPOSITE_COLUMN_VALUE FilterType = 2
	FT_COLUMN_PAGINATION FilterType = 3
)

type ColumnFilter interface {
	Serialize() []byte
	ToFilter() *otsprotocol.Filter
}

type VariantType int32

const (
	Variant_INTEGER VariantType = 0;
	Variant_DOUBLE VariantType = 1;
	//VT_BOOLEAN = 2;
	Variant_STRING VariantType = 3;
)

type ValueTransferRule struct {
	Regex     string
	Cast_type VariantType
}

type SingleColumnCondition struct {
	Comparator        *ComparatorType
	ColumnName        *string
	ColumnValue       interface{} //[]byte
	FilterIfMissing   bool
	LatestVersionOnly bool
	TransferRule      *ValueTransferRule
}

type ReturnType int32

const (
	ReturnType_RT_NONE ReturnType = 0
	ReturnType_RT_PK ReturnType = 1
	ReturnType_RT_AFTER_MODIFY ReturnType = 2
)

type PaginationFilter struct {
	Offset int32
	Limit  int32
}

type CompositeColumnValueFilter struct {
	Operator LogicalOperator
	Filters  []ColumnFilter
}

func (ccvfilter *CompositeColumnValueFilter) Serialize() []byte {
	result, _ := proto.Marshal(ccvfilter.ToFilter())
	return result
}

func (ccvfilter *CompositeColumnValueFilter) ToFilter() *otsprotocol.Filter {
	compositefilter := NewCompositeFilter(ccvfilter.Filters, ccvfilter.Operator)
	compositeFilterToBytes, _ := proto.Marshal(compositefilter)
	filter := new(otsprotocol.Filter)
	filter.Type = otsprotocol.FilterType_FT_COMPOSITE_COLUMN_VALUE.Enum()
	filter.Filter = compositeFilterToBytes
	return filter
}

func (ccvfilter *CompositeColumnValueFilter) AddFilter(filter ColumnFilter) {
	ccvfilter.Filters = append(ccvfilter.Filters, filter)
}

func (condition *SingleColumnCondition) ToFilter() *otsprotocol.Filter {
	singlefilter := NewSingleColumnValueFilter(condition)
	singleFilterToBytes, _ := proto.Marshal(singlefilter)
	filter := new(otsprotocol.Filter)
	filter.Type = otsprotocol.FilterType_FT_SINGLE_COLUMN_VALUE.Enum()
	filter.Filter = singleFilterToBytes
	return filter
}

func (condition *SingleColumnCondition) Serialize() []byte {
	result, _ := proto.Marshal(condition.ToFilter())
	return result
}

func (pageFilter *PaginationFilter) ToFilter() *otsprotocol.Filter {
	compositefilter := NewPaginationFilter(pageFilter)
	compositeFilterToBytes, _ := proto.Marshal(compositefilter)
	filter := new(otsprotocol.Filter)
	filter.Type = otsprotocol.FilterType_FT_COLUMN_PAGINATION.Enum()
	filter.Filter = compositeFilterToBytes
	return filter
}

func (pageFilter *PaginationFilter) Serialize() []byte {
	result, _ := proto.Marshal(pageFilter.ToFilter())
	return result
}

func NewTableOptionWithMaxVersion(maxVersion int) *TableOption {
	tableOption := new(TableOption)
	tableOption.TimeToAlive = -1
	tableOption.MaxVersion = maxVersion
	return tableOption
}

func NewTableOption(timeToAlive int, maxVersion int) *TableOption {
	tableOption := new(TableOption)
	tableOption.TimeToAlive = timeToAlive
	tableOption.MaxVersion = maxVersion
	return tableOption
}

type RowCondition struct {
	RowExistenceExpectation RowExistenceExpectation
	ColumnCondition         ColumnFilter
}

type PutRowChange struct {
	TableName  string
	PrimaryKey *PrimaryKey
	Columns    []AttributeColumn
	Condition  *RowCondition
	ReturnType ReturnType
	TransactionId    *string
}

type PutRowRequest struct {
	PutRowChange *PutRowChange
}

type DeleteRowChange struct {
	TableName  string
	PrimaryKey *PrimaryKey
	Condition  *RowCondition
	TransactionId *string
}

type DeleteRowRequest struct {
	DeleteRowChange *DeleteRowChange
}

type SingleRowQueryCriteria struct {
	ColumnsToGet []string
	TableName    string
	PrimaryKey   *PrimaryKey
	MaxVersion   int32
	TimeRange    *TimeRange
	Filter       ColumnFilter
	StartColumn  *string
	EndColumn    *string
	TransactionId *string
}

type UpdateRowChange struct {
	TableName  string
	PrimaryKey *PrimaryKey
	Columns    []ColumnToUpdate
	Condition  *RowCondition
	TransactionId *string
	ReturnType ReturnType
	ColumnNamesToReturn    []string
}

type UpdateRowRequest struct {
	UpdateRowChange *UpdateRowChange
}

func (rowQueryCriteria *SingleRowQueryCriteria) AddColumnToGet(columnName string) {
	rowQueryCriteria.ColumnsToGet = append(rowQueryCriteria.ColumnsToGet, columnName)
}

func (rowQueryCriteria *SingleRowQueryCriteria) SetStartColumn(columnName string) {
	rowQueryCriteria.StartColumn = &columnName
}

func (rowQueryCriteria *SingleRowQueryCriteria) SetEndtColumn(columnName string) {
	rowQueryCriteria.EndColumn = &columnName
}

func (rowQueryCriteria *SingleRowQueryCriteria) getColumnsToGet() []string {
	return rowQueryCriteria.ColumnsToGet
}

func (rowQueryCriteria *MultiRowQueryCriteria) AddColumnToGet(columnName string) {
	rowQueryCriteria.ColumnsToGet = append(rowQueryCriteria.ColumnsToGet, columnName)
}

func (rowQueryCriteria *RangeRowQueryCriteria) AddColumnToGet(columnName string) {
	rowQueryCriteria.ColumnsToGet = append(rowQueryCriteria.ColumnsToGet, columnName)
}

func (rowQueryCriteria *MultiRowQueryCriteria) AddRow(pk *PrimaryKey) {
	rowQueryCriteria.PrimaryKey = append(rowQueryCriteria.PrimaryKey, pk)
}

type GetRowRequest struct {
	SingleRowQueryCriteria *SingleRowQueryCriteria
}

type MultiRowQueryCriteria struct {
	PrimaryKey   []*PrimaryKey
	ColumnsToGet []string
	TableName    string
	MaxVersion   int
	TimeRange    *TimeRange
	Filter       ColumnFilter
	StartColumn  *string
	EndColumn    *string
}

type BatchGetRowRequest struct {
	MultiRowQueryCriteria []*MultiRowQueryCriteria
}

type ColumnMap struct {
	Columns    map[string][]*AttributeColumn
	columnsKey []string
}

type GetRowResponse struct {
	PrimaryKey           PrimaryKey
	Columns              []*AttributeColumn
	ConsumedCapacityUnit *ConsumedCapacityUnit
	columnMap            *ColumnMap
	ResponseInfo
}

type Error struct {
	Code    string
	Message string
}

type RowResult struct {
	TableName            string
	IsSucceed            bool
	Error                Error
	PrimaryKey           PrimaryKey
	Columns              []*AttributeColumn
	ConsumedCapacityUnit *ConsumedCapacityUnit
	Index                int32
}

type RowChange interface {
	Serialize() []byte
	getOperationType() otsprotocol.OperationType
	getCondition() *otsprotocol.Condition
	GetTableName() string
}

type BatchGetRowResponse struct {
	TableToRowsResult map[string][]RowResult
	ResponseInfo
}

type BatchWriteRowRequest struct {
	RowChangesGroupByTable map[string][]RowChange
}

type BatchWriteRowResponse struct {
	TableToRowsResult map[string][]RowResult
	ResponseInfo
}

type Direction int32

const (
	FORWARD Direction = 0
	BACKWARD Direction = 1
)

type RangeRowQueryCriteria struct {
	TableName       string
	StartPrimaryKey *PrimaryKey
	EndPrimaryKey   *PrimaryKey
	ColumnsToGet    []string
	MaxVersion      int32
	TimeRange       *TimeRange
	Filter          ColumnFilter
	Direction       Direction
	Limit           int32
	StartColumn     *string
	EndColumn       *string
	TransactionId    *string
}

type GetRangeRequest struct {
	RangeRowQueryCriteria *RangeRowQueryCriteria
}

type Row struct {
	PrimaryKey *PrimaryKey
	Columns    []*AttributeColumn
}

type GetRangeResponse struct {
	Rows                 []*Row
	ConsumedCapacityUnit *ConsumedCapacityUnit
	NextStartPrimaryKey  *PrimaryKey
	ResponseInfo
}

type ListStreamRequest struct {
	TableName *string
}

type Stream struct {
	Id           *StreamId
	TableName    *string
	CreationTime int64
}

type ListStreamResponse struct {
	Streams []Stream
	ResponseInfo
}

type StreamSpecification struct {
	EnableStream   bool
	ExpirationTime int32 // must be positive. in hours
}

type StreamDetails struct {
	EnableStream   bool
	StreamId       *StreamId // nil when stream is disabled.
	ExpirationTime int32     // in hours
	LastEnableTime int64     // the last time stream is enabled, in usec
}

type DescribeStreamRequest struct {
	StreamId              *StreamId // required
	InclusiveStartShardId *ShardId  // optional
	ShardLimit            *int32    // optional
}

type DescribeStreamResponse struct {
	StreamId       *StreamId    // required
	ExpirationTime int32        // in hours
	TableName      *string      // required
	CreationTime   int64        // in usec
	Status         StreamStatus // required
	Shards         []*StreamShard
	NextShardId    *ShardId     // optional. nil means "no more shards"
	ResponseInfo
}

type GetShardIteratorRequest struct {
	StreamId  *StreamId // required
	ShardId   *ShardId  // required
	Timestamp *int64
	Token     *string
}

type GetShardIteratorResponse struct {
	ShardIterator *ShardIterator // required
	Token         *string
	ResponseInfo
}

type GetStreamRecordRequest struct {
	ShardIterator *ShardIterator // required
	Limit         *int32         // optional. max records which will reside in response
}

type GetStreamRecordResponse struct {
	Records           []*StreamRecord
	NextShardIterator *ShardIterator // optional. an indicator to be used to read more records in this shard
	ResponseInfo
}

type ComputeSplitPointsBySizeRequest struct {
	TableName string
	SplitSize int64
}

type ComputeSplitPointsBySizeResponse struct {
	SchemaEntry []*PrimaryKeySchema
	Splits      []*Split
	ResponseInfo
}

type Split struct {
	LowerBound *PrimaryKey
	UpperBound *PrimaryKey
	Location   string
}

type StreamId string
type ShardId string
type ShardIterator string
type StreamStatus int

const (
	SS_Enabling StreamStatus = iota
	SS_Active
)

/*
 * Shards are possibly splitted into two or merged from two.
 * After splitting, both newly generated shards have the same FatherShard.
 * After merging, the newly generated shard have both FatherShard and MotherShard.
 */
type StreamShard struct {
	SelfShard   *ShardId // required
	FatherShard *ShardId // optional
	MotherShard *ShardId // optional
}

type StreamRecord struct {
	Type       ActionType
	Info       *RecordSequenceInfo // required
	PrimaryKey *PrimaryKey         // required
	Columns    []*RecordColumn
}

func (this *StreamRecord) String() string {
	return fmt.Sprintf(
		"{\"Type\":%s, \"PrimaryKey\":%s, \"Info\":%s, \"Columns\":%s}",
		this.Type,
		*this.PrimaryKey,
		this.Info,
		this.Columns)
}

type ActionType int

const (
	AT_Put ActionType = iota
	AT_Update
	AT_Delete
)

func (this ActionType) String() string {
	switch this {
	case AT_Put:
		return "\"PutRow\""
	case AT_Update:
		return "\"UpdateRow\""
	case AT_Delete:
		return "\"DeleteRow\""
	default:
		panic(fmt.Sprintf("unknown action type: %d", int(this)))
	}
}

type RecordSequenceInfo struct {
	Epoch     int32
	Timestamp int64
	RowIndex  int32
}

func (this *RecordSequenceInfo) String() string {
	return fmt.Sprintf(
		"{\"Epoch\":%d, \"Timestamp\": %d, \"RowIndex\": %d}",
		this.Epoch,
		this.Timestamp,
		this.RowIndex)
}

type RecordColumn struct {
	Type      RecordColumnType
	Name      *string     // required
	Value     interface{} // optional. present when Type is RCT_Put
	Timestamp *int64      // optional, in msec. present when Type is RCT_Put or RCT_DeleteOneVersion
}

func (this *RecordColumn) String() string {
	xs := make([]string, 0)
	xs = append(xs, fmt.Sprintf("\"Name\":%s", strconv.Quote(*this.Name)))
	switch this.Type {
	case RCT_DeleteAllVersions:
		xs = append(xs, "\"Type\":\"DeleteAllVersions\"")
	case RCT_DeleteOneVersion:
		xs = append(xs, "\"Type\":\"DeleteOneVersion\"")
		xs = append(xs, fmt.Sprintf("\"Timestamp\":%d", *this.Timestamp))
	case RCT_Put:
		xs = append(xs, "\"Type\":\"Put\"")
		xs = append(xs, fmt.Sprintf("\"Timestamp\":%d", *this.Timestamp))
		xs = append(xs, fmt.Sprintf("\"Value\":%s", this.Value))
	}
	return fmt.Sprintf("{%s}", strings.Join(xs, ", "))
}

type RecordColumnType int

const (
	RCT_Put RecordColumnType = iota
	RCT_DeleteOneVersion
	RCT_DeleteAllVersions
)

type IndexMeta struct {
	IndexName      string
	Primarykey     []string
	DefinedColumns []string
	IndexType      IndexType
}

type DefinedColumnSchema struct {
	Name       string
	ColumnType DefinedColumnType
}

type IndexType int32

const (
	IT_GLOBAL_INDEX IndexType = 1
	IT_LOCAL_INDEX IndexType = 2
)

type DefinedColumnType int32

const (
	/**
	 * 64位整数。
	 */
	DefinedColumn_INTEGER DefinedColumnType = 1

	/**
	 * 浮点数。
	 */
	DefinedColumn_DOUBLE DefinedColumnType = 2

	/**
	 * 布尔值。
	 */
	DefinedColumn_BOOLEAN DefinedColumnType = 3

	/**
	 * 字符串。
	 */
	DefinedColumn_STRING DefinedColumnType = 4

	/**
	 * BINARY。
	 */
	DefinedColumn_BINARY DefinedColumnType = 5
)

type StartLocalTransactionRequest struct {
	PrimaryKey *PrimaryKey
	TableName string
}

type StartLocalTransactionResponse struct {
	TransactionId    *string
	ResponseInfo
}

type CommitTransactionRequest struct {
	TransactionId    *string
}

type CommitTransactionResponse struct {
	ResponseInfo
}

type AbortTransactionRequest struct {
	TransactionId    *string
}

type AbortTransactionResponse struct {
	ResponseInfo
}