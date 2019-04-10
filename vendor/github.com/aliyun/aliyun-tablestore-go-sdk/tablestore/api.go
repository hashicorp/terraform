package tablestore

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"net"
	"net/http"
	"time"
	"io"
	"strings"
)

const (
	userAgent = "aliyun-tablestore-sdk-golang/4.0.2"

	createTableUri                     = "/CreateTable"
	listTableUri                       = "/ListTable"
	deleteTableUri                     = "/DeleteTable"
	describeTableUri                   = "/DescribeTable"
	updateTableUri                     = "/UpdateTable"
	putRowUri                          = "/PutRow"
	deleteRowUri                       = "/DeleteRow"
	getRowUri                          = "/GetRow"
	updateRowUri                       = "/UpdateRow"
	batchGetRowUri                     = "/BatchGetRow"
	batchWriteRowUri                   = "/BatchWriteRow"
	getRangeUri                        = "/GetRange"
	listStreamUri                      = "/ListStream"
	describeStreamUri                  = "/DescribeStream"
	getShardIteratorUri                = "/GetShardIterator"
	getStreamRecordUri                 = "/GetStreamRecord"
	computeSplitPointsBySizeRequestUri = "/ComputeSplitPointsBySize"
	searchUri                          = "/Search"
	createSearchIndexUri               = "/CreateSearchIndex"
	listSearchIndexUri                 = "/ListSearchIndex"
	deleteSearchIndexUri               = "/DeleteSearchIndex"
	describeSearchIndexUri             = "/DescribeSearchIndex"

	createIndexUri                     = "/CreateIndex"
	dropIndexUri                       = "/DropIndex"

	createlocaltransactionuri          = "/StartLocalTransaction"
	committransactionuri               = "/CommitTransaction"
	aborttransactionuri                = "/AbortTransaction"
)

// Constructor: to create the client of TableStore service.
// 构造函数：创建表格存储服务的客户端。
//
// @param endPoint The address of TableStore service. 表格存储服务地址。
// @param instanceName
// @param accessId The Access ID. 用于标示用户的ID。
// @param accessKey The Access Key. 用于签名和验证的密钥。
// @param options set client config
func NewClient(endPoint, instanceName, accessKeyId, accessKeySecret string, options ...ClientOption) *TableStoreClient {
	client := NewClientWithConfig(endPoint, instanceName, accessKeyId, accessKeySecret, "", nil)
	// client options parse
	for _, option := range options {
		option(client)
	}

	return client
}

type GetHttpClient func() IHttpClient

var currentGetHttpClientFunc GetHttpClient = func() IHttpClient {
	return &TableStoreHttpClient{}
}

// Constructor: to create the client of OTS service. 传入config
// 构造函数：创建OTS服务的客户端。
func NewClientWithConfig(endPoint, instanceName, accessKeyId, accessKeySecret string, securityToken string, config *TableStoreConfig) *TableStoreClient {
	tableStoreClient := new(TableStoreClient)
	tableStoreClient.endPoint = endPoint
	tableStoreClient.instanceName = instanceName
	tableStoreClient.accessKeyId = accessKeyId
	tableStoreClient.accessKeySecret = accessKeySecret
	tableStoreClient.securityToken = securityToken
	if config == nil {
		config = NewDefaultTableStoreConfig()
	}
	tableStoreClient.config = config
	tableStoreTransportProxy := &http.Transport{
		MaxIdleConnsPerHost: config.MaxIdleConnections,
		Dial: (&net.Dialer{
			Timeout: config.HTTPTimeout.ConnectionTimeout,
		}).Dial,
	}

	tableStoreClient.httpClient = currentGetHttpClientFunc()

	httpClient := &http.Client{
		Transport: tableStoreTransportProxy,
		Timeout:   tableStoreClient.config.HTTPTimeout.RequestTimeout,
	}
	tableStoreClient.httpClient.New(httpClient)

	tableStoreClient.random = rand.New(rand.NewSource(time.Now().Unix()))

	return tableStoreClient
}

// 请求服务端
func (tableStoreClient *TableStoreClient) doRequestWithRetry(uri string, req, resp proto.Message, responseInfo *ResponseInfo) error {
	end := time.Now().Add(tableStoreClient.config.MaxRetryTime)
	url := fmt.Sprintf("%s%s", tableStoreClient.endPoint, uri)
	/* request body */
	var body []byte
	var err error
	if req != nil {
		body, err = proto.Marshal(req)
		if err != nil {
			return err
		}
	} else {
		body = nil
	}

	var value int64
	var i uint
	var respBody []byte
	var requestId string
	for i = 0; ; i++ {
		respBody, err, requestId = tableStoreClient.doRequest(url, uri, body, resp)
		responseInfo.RequestId = requestId

		if err == nil {
			break
		} else {
			value = getNextPause(tableStoreClient, err, i, end, value, uri)

			// fmt.Println("hit retry", uri, err, *e.Code, value)
			if value <= 0 {
				return err
			}

			time.Sleep(time.Duration(value) * time.Millisecond)
		}
	}

	if respBody == nil || len(respBody) == 0 {
		return nil
	}

	err = proto.Unmarshal(respBody, resp)
	if err != nil {
		return fmt.Errorf("decode resp failed: %s", err)
	}

	return nil
}

func getNextPause(tableStoreClient *TableStoreClient, err error, count uint, end time.Time, lastInterval int64, action string) int64 {
	if tableStoreClient.config.RetryTimes <= count || time.Now().After(end) {
		return 0
	}
	var retry bool
	if otsErr, ok := err.(*OtsError); ok {
		retry = shouldRetry(otsErr.Code, otsErr.Message, action)
	} else {
		if err == io.EOF || err == io.ErrUnexpectedEOF || //retry on special net error contains EOF or reset
			strings.Contains(err.Error(), io.EOF.Error()) ||
			strings.Contains(err.Error(), "Connection reset by peer") ||
			strings.Contains(err.Error(), "connection reset by peer") {
			retry = true
		} else if nErr, ok := err.(net.Error); ok {
			retry = nErr.Temporary()
		}
	}

	if retry {
		value := lastInterval*2 + tableStoreClient.random.Int63n(DefaultRetryInterval-1) + 1
		if value > MaxRetryInterval {
			value =  MaxRetryInterval
		}

		return value
	}
	return 0
}

func shouldRetry(errorCode string, errorMsg string, action string) bool {
	if retryNotMatterActions(errorCode, errorMsg) == true {
		return true
	}

	if isIdempotent(action) &&
		(errorCode == STORAGE_TIMEOUT || errorCode == INTERNAL_SERVER_ERROR || errorCode == SERVER_UNAVAILABLE) {
		return true
	}
	return false
}

func retryNotMatterActions(errorCode string, errorMsg string) bool {
	if errorCode == ROW_OPERATION_CONFLICT || errorCode == NOT_ENOUGH_CAPACITY_UNIT ||
		errorCode == TABLE_NOT_READY || errorCode == PARTITION_UNAVAILABLE ||
		errorCode == SERVER_BUSY || errorCode == STORAGE_SERVER_BUSY || (errorCode == QUOTA_EXHAUSTED && errorMsg == "Too frequent table operations.") {
		return true
	} else {
		return false
	}
}

func isIdempotent(action string) bool {
	if action == batchGetRowUri || action == describeTableUri ||
		action == getRangeUri || action == getRowUri ||
		action == listTableUri || action == listStreamUri ||
			action == getStreamRecordUri || action == describeStreamUri {
		return true
	} else {
		return false
	}
}

func (tableStoreClient *TableStoreClient) doRequest(url string, uri string, body []byte, resp proto.Message) ([]byte, error, string) {
	hreq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err, ""
	}
	/* set headers */
	hreq.Header.Set("User-Agent", userAgent)

	date := time.Now().UTC().Format(xOtsDateFormat)

	hreq.Header.Set(xOtsDate, date)
	hreq.Header.Set(xOtsApiversion, ApiVersion)
	hreq.Header.Set(xOtsAccesskeyid, tableStoreClient.accessKeyId)
	hreq.Header.Set(xOtsInstanceName, tableStoreClient.instanceName)

	md5Byte := md5.Sum(body)
	md5Base64 := base64.StdEncoding.EncodeToString(md5Byte[:16])
	hreq.Header.Set(xOtsContentmd5, md5Base64)

	otshead := createOtsHeaders(tableStoreClient.accessKeySecret)
	otshead.set(xOtsDate, date)
	otshead.set(xOtsApiversion, ApiVersion)
	otshead.set(xOtsAccesskeyid, tableStoreClient.accessKeyId)
	if tableStoreClient.securityToken != "" {
		hreq.Header.Set(xOtsHeaderStsToken, tableStoreClient.securityToken)
		otshead.set(xOtsHeaderStsToken, tableStoreClient.securityToken)
	}
	otshead.set(xOtsContentmd5, md5Base64)
	otshead.set(xOtsInstanceName, tableStoreClient.instanceName)
	sign, err := otshead.signature(uri, "POST", tableStoreClient.accessKeySecret)

	if err != nil {
		return nil, err, ""
	}
	hreq.Header.Set(xOtsSignature, sign)

	/* end set headers */
	return tableStoreClient.postReq(hreq, url)
}

// table API
// Create a table with the CreateTableRequest, in which the table name and
// primary keys are required.
// 根据CreateTableRequest创建一个表，其中表名和主健列是必选项
//
// @param request of CreateTableRequest.
// @return Void. 无返回值。
func (tableStoreClient *TableStoreClient) CreateTable(request *CreateTableRequest) (*CreateTableResponse, error) {
	if len(request.TableMeta.TableName) > maxTableNameLength {
		return nil, errTableNameTooLong(request.TableMeta.TableName)
	}

	if len(request.TableMeta.SchemaEntry) > maxPrimaryKeyNum {
		return nil, errPrimaryKeyTooMuch
	}

	if len(request.TableMeta.SchemaEntry) == 0 {
		return nil, errCreateTableNoPrimaryKey
	}

	req := new(otsprotocol.CreateTableRequest)
	req.TableMeta = new(otsprotocol.TableMeta)
	req.TableMeta.TableName = proto.String(request.TableMeta.TableName)

	if len(request.TableMeta.DefinedColumns) > 0 {
		for _, value := range request.TableMeta.DefinedColumns {
			req.TableMeta.DefinedColumn = append(req.TableMeta.DefinedColumn, &otsprotocol.DefinedColumnSchema{Name: &value.Name, Type: value.ColumnType.ConvertToPbDefinedColumnType().Enum() })
		}
	}

	if len(request.IndexMetas) > 0 {
		for _, value := range request.IndexMetas {
			req.IndexMetas = append(req.IndexMetas, value.ConvertToPbIndexMeta())
		}
	}

	for _, key := range request.TableMeta.SchemaEntry {
		keyType := otsprotocol.PrimaryKeyType(*key.Type)
		if key.Option != nil {
			keyOption := otsprotocol.PrimaryKeyOption(*key.Option)
			req.TableMeta.PrimaryKey = append(req.TableMeta.PrimaryKey, &otsprotocol.PrimaryKeySchema{Name: key.Name, Type: &keyType, Option: &keyOption})
		} else {
			req.TableMeta.PrimaryKey = append(req.TableMeta.PrimaryKey, &otsprotocol.PrimaryKeySchema{Name: key.Name, Type: &keyType})
		}
	}

	req.ReservedThroughput = new(otsprotocol.ReservedThroughput)
	req.ReservedThroughput.CapacityUnit = new(otsprotocol.CapacityUnit)
	req.ReservedThroughput.CapacityUnit.Read = proto.Int32(int32(request.ReservedThroughput.Readcap))
	req.ReservedThroughput.CapacityUnit.Write = proto.Int32(int32(request.ReservedThroughput.Writecap))

	req.TableOptions = new(otsprotocol.TableOptions)
	req.TableOptions.TimeToLive = proto.Int32(int32(request.TableOption.TimeToAlive))
	req.TableOptions.MaxVersions = proto.Int32(int32(request.TableOption.MaxVersion))

	if request.StreamSpec != nil {
		var ss otsprotocol.StreamSpecification
		if request.StreamSpec.EnableStream {
			ss = otsprotocol.StreamSpecification{
				EnableStream:   &request.StreamSpec.EnableStream,
				ExpirationTime: &request.StreamSpec.ExpirationTime}
		} else {
			ss = otsprotocol.StreamSpecification{
				EnableStream: &request.StreamSpec.EnableStream}
		}

		req.StreamSpec = &ss
	}

	resp := new(otsprotocol.CreateTableResponse)
	response := &CreateTableResponse{}
	if err := tableStoreClient.doRequestWithRetry(createTableUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	return response, nil
}

func (tableStoreClient *TableStoreClient) CreateIndex(request *CreateIndexRequest) (*CreateIndexResponse, error) {
	if len(request.MainTableName) > maxTableNameLength {
		return nil, errTableNameTooLong(request.MainTableName)
	}

	req := new(otsprotocol.CreateIndexRequest)
	req.IndexMeta = request.IndexMeta.ConvertToPbIndexMeta()
	req.IncludeBaseData = proto.Bool(request.IncludeBaseData)
	req.MainTableName = proto.String(request.MainTableName)

	resp := new(otsprotocol.CreateIndexResponse)
	response := &CreateIndexResponse{}
	if err := tableStoreClient.doRequestWithRetry(createIndexUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	return response, nil
}

func (tableStoreClient *TableStoreClient) DeleteIndex(request *DeleteIndexRequest) (*DeleteIndexResponse, error) {
	if len(request.MainTableName) > maxTableNameLength {
		return nil, errTableNameTooLong(request.MainTableName)
	}

	req := new(otsprotocol.DropIndexRequest)
	req.IndexName = proto.String(request.IndexName)
	req.MainTableName = proto.String(request.MainTableName)

	resp := new(otsprotocol.DropIndexResponse)
	response := &DeleteIndexResponse{}
	if err := tableStoreClient.doRequestWithRetry(dropIndexUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	return response, nil
}

// List all tables. If done, all table names will be returned.
// 列出所有的表，如果操作成功，将返回所有表的名称。
//
// @param tableNames The returned table names. 返回的表名集合。
// @return Void. 无返回值。
func (tableStoreClient *TableStoreClient) ListTable() (*ListTableResponse, error) {
	resp := new(otsprotocol.ListTableResponse)
	response := &ListTableResponse{}
	if err := tableStoreClient.doRequestWithRetry(listTableUri, nil, resp, &response.ResponseInfo); err != nil {
		return response, err
	}

	response.TableNames = resp.TableNames
	return response, nil
}

// Delete a table and all its views will be deleted.
// 删除一个表
//
// @param tableName The table name. 表名。
// @return Void. 无返回值。
func (tableStoreClient *TableStoreClient) DeleteTable(request *DeleteTableRequest) (*DeleteTableResponse, error) {
	req := new(otsprotocol.DeleteTableRequest)
	req.TableName = proto.String(request.TableName)

	response := &DeleteTableResponse{}
	if err := tableStoreClient.doRequestWithRetry(deleteTableUri, req, nil, &response.ResponseInfo); err != nil {
		return nil, err
	}
	return response, nil
}

// Query the tablemeta, tableoption and reservedthroughtputdetails
// @param DescribeTableRequest
// @param DescribeTableResponse
func (tableStoreClient *TableStoreClient) DescribeTable(request *DescribeTableRequest) (*DescribeTableResponse, error) {
	req := new(otsprotocol.DescribeTableRequest)
	req.TableName = proto.String(request.TableName)

	resp := new(otsprotocol.DescribeTableResponse)
	response := new(DescribeTableResponse)

	if err := tableStoreClient.doRequestWithRetry(describeTableUri, req, resp, &response.ResponseInfo); err != nil {
		return &DescribeTableResponse{}, err
	}

	response.ReservedThroughput = &ReservedThroughput{Readcap: int(*(resp.ReservedThroughputDetails.CapacityUnit.Read)), Writecap: int(*(resp.ReservedThroughputDetails.CapacityUnit.Write))}

	responseTableMeta := new(TableMeta)
	responseTableMeta.TableName = *resp.TableMeta.TableName

	for _, key := range resp.TableMeta.PrimaryKey {
		keyType := PrimaryKeyType(*key.Type)

		// enable it when we support kep option in describe table
		if key.Option != nil {
			keyOption := PrimaryKeyOption(*key.Option)
			responseTableMeta.SchemaEntry = append(responseTableMeta.SchemaEntry, &PrimaryKeySchema{Name: key.Name, Type: &keyType, Option: &keyOption})
		} else {
			responseTableMeta.SchemaEntry = append(responseTableMeta.SchemaEntry, &PrimaryKeySchema{Name: key.Name, Type: &keyType})
		}
	}
	response.TableMeta = responseTableMeta
	response.TableOption = &TableOption{TimeToAlive: int(*resp.TableOptions.TimeToLive), MaxVersion: int(*resp.TableOptions.MaxVersions)}
	if resp.StreamDetails != nil && *resp.StreamDetails.EnableStream {
		response.StreamDetails = &StreamDetails{
			EnableStream:   *resp.StreamDetails.EnableStream,
			StreamId:       (*StreamId)(resp.StreamDetails.StreamId),
			ExpirationTime: *resp.StreamDetails.ExpirationTime,
			LastEnableTime: *resp.StreamDetails.LastEnableTime}
	} else {
		response.StreamDetails = &StreamDetails{
			EnableStream: false}
	}

	for _, meta := range resp.IndexMetas {
		response.IndexMetas = append(response.IndexMetas, ConvertPbIndexMetaToIndexMeta(meta))
	}

	return response, nil
}

// Update the table info includes tableoptions and reservedthroughput
// @param UpdateTableRequest
// @param UpdateTableResponse
func (tableStoreClient *TableStoreClient) UpdateTable(request *UpdateTableRequest) (*UpdateTableResponse, error) {
	req := new(otsprotocol.UpdateTableRequest)
	req.TableName = proto.String(request.TableName)

	if request.ReservedThroughput != nil {
		req.ReservedThroughput = new(otsprotocol.ReservedThroughput)
		req.ReservedThroughput.CapacityUnit = new(otsprotocol.CapacityUnit)
		req.ReservedThroughput.CapacityUnit.Read = proto.Int32(int32(request.ReservedThroughput.Readcap))
		req.ReservedThroughput.CapacityUnit.Write = proto.Int32(int32(request.ReservedThroughput.Writecap))
	}

	if request.TableOption != nil {
		req.TableOptions = new(otsprotocol.TableOptions)
		req.TableOptions.TimeToLive = proto.Int32(int32(request.TableOption.TimeToAlive))
		req.TableOptions.MaxVersions = proto.Int32(int32(request.TableOption.MaxVersion))
	}

	if request.StreamSpec != nil {
		req.StreamSpec = &otsprotocol.StreamSpecification{
			EnableStream:   &request.StreamSpec.EnableStream,
			ExpirationTime: &request.StreamSpec.ExpirationTime}
	}

	resp := new(otsprotocol.UpdateTableResponse)
	response := new(UpdateTableResponse)

	if err := tableStoreClient.doRequestWithRetry(updateTableUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	response.ReservedThroughput = &ReservedThroughput{
		Readcap:  int(*(resp.ReservedThroughputDetails.CapacityUnit.Read)),
		Writecap: int(*(resp.ReservedThroughputDetails.CapacityUnit.Write))}
	response.TableOption = &TableOption{
		TimeToAlive: int(*resp.TableOptions.TimeToLive),
		MaxVersion:  int(*resp.TableOptions.MaxVersions)}
	if *resp.StreamDetails.EnableStream {
		response.StreamDetails = &StreamDetails{
			EnableStream:   *resp.StreamDetails.EnableStream,
			StreamId:       (*StreamId)(resp.StreamDetails.StreamId),
			ExpirationTime: *resp.StreamDetails.ExpirationTime,
			LastEnableTime: *resp.StreamDetails.LastEnableTime}
	} else {
		response.StreamDetails = &StreamDetails{
			EnableStream: false}
	}
	return response, nil
}

// Put or update a row in a table. The operation is determined by CheckingType,
// which has three options: NO, UPDATE, INSERT. The transaction id is optional.
// 插入或更新行数据。操作针对数据的存在性包含三种检查类型：NO(不检查)，UPDATE
// （更新，数据必须存在）和INSERT（插入，数据必须不存在）。事务ID是可选项。
//
// @param builder The builder for putting a row. 插入或更新数据的Builder。
// @return Void. 无返回值。
func (tableStoreClient *TableStoreClient) PutRow(request *PutRowRequest) (*PutRowResponse, error) {
	if request == nil {
		return nil, nil
	}

	if request.PutRowChange == nil {
		return nil, nil
	}

	req := new(otsprotocol.PutRowRequest)
	req.TableName = proto.String(request.PutRowChange.TableName)
	req.Row = request.PutRowChange.Serialize()

	condition := new(otsprotocol.Condition)
	condition.RowExistence = request.PutRowChange.Condition.buildCondition()
	if request.PutRowChange.Condition.ColumnCondition != nil {
		condition.ColumnCondition = request.PutRowChange.Condition.ColumnCondition.Serialize()
	}

	if request.PutRowChange.ReturnType == ReturnType_RT_PK {
		content := otsprotocol.ReturnContent{ReturnType: otsprotocol.ReturnType_RT_PK.Enum()}
		req.ReturnContent = &content
	}

	if request.PutRowChange.TransactionId != nil {
		req.TransactionId = request.PutRowChange.TransactionId
	}

	req.Condition = condition

	resp := new(otsprotocol.PutRowResponse)
	response := &PutRowResponse{}
	if err := tableStoreClient.doRequestWithRetry(putRowUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	response.ConsumedCapacityUnit = &ConsumedCapacityUnit{}
	response.ConsumedCapacityUnit.Read = *resp.Consumed.CapacityUnit.Read
	response.ConsumedCapacityUnit.Write = *resp.Consumed.CapacityUnit.Write

	if request.PutRowChange.ReturnType == ReturnType_RT_PK {
		rows, err := readRowsWithHeader(bytes.NewReader(resp.Row))
		if err != nil {
			return response, err
		}

		for _, pk := range rows[0].primaryKey {
			pkColumn := &PrimaryKeyColumn{ColumnName: string(pk.cellName), Value: pk.cellValue.Value}
			response.PrimaryKey.PrimaryKeys = append(response.PrimaryKey.PrimaryKeys, pkColumn)
		}
	}

	return response, nil
}

// Delete row with pk
// @param DeleteRowRequest
func (tableStoreClient *TableStoreClient) DeleteRow(request *DeleteRowRequest) (*DeleteRowResponse, error) {
	req := new(otsprotocol.DeleteRowRequest)
	req.TableName = proto.String(request.DeleteRowChange.TableName)
	req.Condition = request.DeleteRowChange.getCondition()
	req.PrimaryKey = request.DeleteRowChange.PrimaryKey.Build(true)

	if request.DeleteRowChange.TransactionId != nil {
		req.TransactionId = request.DeleteRowChange.TransactionId
	}

	resp := new(otsprotocol.DeleteRowResponse)
	response := &DeleteRowResponse{}
	if err := tableStoreClient.doRequestWithRetry(deleteRowUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	response.ConsumedCapacityUnit = &ConsumedCapacityUnit{}
	response.ConsumedCapacityUnit.Read = *resp.Consumed.CapacityUnit.Read
	response.ConsumedCapacityUnit.Write = *resp.Consumed.CapacityUnit.Write
	return response, nil
}

// row API
// Get the data of a row or some columns.
//
// @param getrowrequest
func (tableStoreClient *TableStoreClient) GetRow(request *GetRowRequest) (*GetRowResponse, error) {
	req := new(otsprotocol.GetRowRequest)
	resp := new(otsprotocol.GetRowResponse)

	req.TableName = proto.String(request.SingleRowQueryCriteria.TableName)

	if (request.SingleRowQueryCriteria.getColumnsToGet() != nil) && len(request.SingleRowQueryCriteria.getColumnsToGet()) > 0 {
		req.ColumnsToGet = request.SingleRowQueryCriteria.getColumnsToGet()
	}

	req.PrimaryKey = request.SingleRowQueryCriteria.PrimaryKey.Build(false)

	if request.SingleRowQueryCriteria.MaxVersion != 0 {
		req.MaxVersions = proto.Int32(int32(request.SingleRowQueryCriteria.MaxVersion))
	}

	if request.SingleRowQueryCriteria.TransactionId != nil {
		req.TransactionId = request.SingleRowQueryCriteria.TransactionId
	}

	if request.SingleRowQueryCriteria.StartColumn != nil {
		req.StartColumn = request.SingleRowQueryCriteria.StartColumn
	}

	if request.SingleRowQueryCriteria.EndColumn != nil {
		req.EndColumn = request.SingleRowQueryCriteria.EndColumn
	}

	if request.SingleRowQueryCriteria.TimeRange != nil {
		if request.SingleRowQueryCriteria.TimeRange.Specific != 0 {
			req.TimeRange = &otsprotocol.TimeRange{SpecificTime: proto.Int64(request.SingleRowQueryCriteria.TimeRange.Specific)}
		} else {
			req.TimeRange = &otsprotocol.TimeRange{StartTime: proto.Int64(request.SingleRowQueryCriteria.TimeRange.Start), EndTime: proto.Int64(request.SingleRowQueryCriteria.TimeRange.End)}
		}
	} else if request.SingleRowQueryCriteria.MaxVersion == 0 {
		return nil, errInvalidInput
	}

	if request.SingleRowQueryCriteria.Filter != nil {
		req.Filter = request.SingleRowQueryCriteria.Filter.Serialize()
	}

	response := &GetRowResponse{ConsumedCapacityUnit: &ConsumedCapacityUnit{}}
	if err := tableStoreClient.doRequestWithRetry(getRowUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	response.ConsumedCapacityUnit.Read = *resp.Consumed.CapacityUnit.Read
	response.ConsumedCapacityUnit.Write = *resp.Consumed.CapacityUnit.Write

	if len(resp.Row) == 0 {
		return response, nil
	}

	rows, err := readRowsWithHeader(bytes.NewReader(resp.Row))
	if err != nil {
		return nil, err
	}

	for _, pk := range rows[0].primaryKey {
		pkColumn := &PrimaryKeyColumn{ColumnName: string(pk.cellName), Value: pk.cellValue.Value}
		response.PrimaryKey.PrimaryKeys = append(response.PrimaryKey.PrimaryKeys, pkColumn)
	}

	for _, cell := range rows[0].cells {
		dataColumn := &AttributeColumn{ColumnName: string(cell.cellName), Value: cell.cellValue.Value, Timestamp: cell.cellTimestamp}
		response.Columns = append(response.Columns, dataColumn)
	}

	return response, nil
}

// Update row
// @param UpdateRowRequest
func (tableStoreClient *TableStoreClient) UpdateRow(request *UpdateRowRequest) (*UpdateRowResponse, error) {
	req := new(otsprotocol.UpdateRowRequest)
	resp := new(otsprotocol.UpdateRowResponse)

	req.TableName = proto.String(request.UpdateRowChange.TableName)
	req.Condition = request.UpdateRowChange.getCondition()
	req.RowChange = request.UpdateRowChange.Serialize()
	if request.UpdateRowChange.TransactionId != nil {
		req.TransactionId = request.UpdateRowChange.TransactionId
	}

	response := &UpdateRowResponse{ConsumedCapacityUnit: &ConsumedCapacityUnit{}}

	if request.UpdateRowChange.ReturnType == ReturnType_RT_AFTER_MODIFY {
		content := otsprotocol.ReturnContent{ReturnType: otsprotocol.ReturnType_RT_AFTER_MODIFY.Enum()}
		for _, column := range request.UpdateRowChange.ColumnNamesToReturn {
			content.ReturnColumnNames = append(content.ReturnColumnNames, column)
		}
		req.ReturnContent = &content
	}

	if err := tableStoreClient.doRequestWithRetry(updateRowUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	if request.UpdateRowChange.ReturnType == ReturnType_RT_AFTER_MODIFY {
		plainbufferRow, err := readRowsWithHeader(bytes.NewReader(resp.Row))
		if err != nil {
			return response, err
		}
		for _, cell := range plainbufferRow[0].cells {
			fmt.Println(cell.cellName)
			attribute := &AttributeColumn{ColumnName: string(cell.cellName), Value: cell.cellValue.Value, Timestamp: cell.cellTimestamp}
			response.Columns = append(response.Columns, attribute)
		}
	}

	response.ConsumedCapacityUnit.Read = *resp.Consumed.CapacityUnit.Read
	response.ConsumedCapacityUnit.Write = *resp.Consumed.CapacityUnit.Write
	return response, nil
}

// Batch Get Row
// @param BatchGetRowRequest
func (tableStoreClient *TableStoreClient) BatchGetRow(request *BatchGetRowRequest) (*BatchGetRowResponse, error) {
	req := new(otsprotocol.BatchGetRowRequest)

	var tablesInBatch []*otsprotocol.TableInBatchGetRowRequest

	for _, Criteria := range request.MultiRowQueryCriteria {
		table := new(otsprotocol.TableInBatchGetRowRequest)
		table.TableName = proto.String(Criteria.TableName)
		table.ColumnsToGet = Criteria.ColumnsToGet

		if Criteria.StartColumn != nil {
			table.StartColumn = Criteria.StartColumn
		}

		if Criteria.EndColumn != nil {
			table.EndColumn = Criteria.EndColumn
		}

		if Criteria.Filter != nil {
			table.Filter = Criteria.Filter.Serialize()
		}

		if Criteria.MaxVersion != 0 {
			table.MaxVersions = proto.Int32(int32(Criteria.MaxVersion))
		}

		if Criteria.TimeRange != nil {
			if Criteria.TimeRange.Specific != 0 {
				table.TimeRange = &otsprotocol.TimeRange{SpecificTime: proto.Int64(Criteria.TimeRange.Specific)}
			} else {
				table.TimeRange = &otsprotocol.TimeRange{StartTime: proto.Int64(Criteria.TimeRange.Start), EndTime: proto.Int64(Criteria.TimeRange.End)}
			}
		} else if Criteria.MaxVersion == 0 {
			return nil, errInvalidInput
		}

		for _, pk := range Criteria.PrimaryKey {
			pkWithBytes := pk.Build(false)
			table.PrimaryKey = append(table.PrimaryKey, pkWithBytes)
		}

		tablesInBatch = append(tablesInBatch, table)
	}

	req.Tables = tablesInBatch
	resp := new(otsprotocol.BatchGetRowResponse)

	response := &BatchGetRowResponse{TableToRowsResult: make(map[string][]RowResult)}
	if err := tableStoreClient.doRequestWithRetry(batchGetRowUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	for _, table := range resp.Tables {
		index := int32(0)
		for _, row := range table.Rows {
			rowResult := &RowResult{TableName: *table.TableName, IsSucceed: *row.IsOk, ConsumedCapacityUnit: &ConsumedCapacityUnit{}, Index: index}
			index++
			if *row.IsOk == false {
				rowResult.Error = Error{Code: *row.Error.Code, Message: *row.Error.Message}
			} else {
				// len == 0 means row not exist
				if len(row.Row) > 0 {
					rows, err := readRowsWithHeader(bytes.NewReader(row.Row))
					if err != nil {
						return nil, err
					}

					for _, pk := range rows[0].primaryKey {
						pkColumn := &PrimaryKeyColumn{ColumnName: string(pk.cellName), Value: pk.cellValue.Value}
						rowResult.PrimaryKey.PrimaryKeys = append(rowResult.PrimaryKey.PrimaryKeys, pkColumn)
					}

					for _, cell := range rows[0].cells {
						dataColumn := &AttributeColumn{ColumnName: string(cell.cellName), Value: cell.cellValue.Value, Timestamp: cell.cellTimestamp}
						rowResult.Columns = append(rowResult.Columns, dataColumn)
					}
				}

				rowResult.ConsumedCapacityUnit.Read = *row.Consumed.CapacityUnit.Read
				rowResult.ConsumedCapacityUnit.Write = *row.Consumed.CapacityUnit.Write
			}

			response.TableToRowsResult[*table.TableName] = append(response.TableToRowsResult[*table.TableName], *rowResult)
		}

	}
	return response, nil
}

// Batch Write Row
// @param BatchWriteRowRequest
func (tableStoreClient *TableStoreClient) BatchWriteRow(request *BatchWriteRowRequest) (*BatchWriteRowResponse, error) {
	req := new(otsprotocol.BatchWriteRowRequest)

	var tablesInBatch []*otsprotocol.TableInBatchWriteRowRequest

	for key, value := range request.RowChangesGroupByTable {
		table := new(otsprotocol.TableInBatchWriteRowRequest)
		table.TableName = proto.String(key)

		for _, row := range value {
			rowInBatch := &otsprotocol.RowInBatchWriteRowRequest{}
			rowInBatch.Condition = row.getCondition()
			rowInBatch.RowChange = row.Serialize()
			rowInBatch.Type = row.getOperationType().Enum()
			table.Rows = append(table.Rows, rowInBatch)
		}

		tablesInBatch = append(tablesInBatch, table)
	}

	req.Tables = tablesInBatch

	resp := new(otsprotocol.BatchWriteRowResponse)
	response := &BatchWriteRowResponse{TableToRowsResult: make(map[string][]RowResult)}

	if err := tableStoreClient.doRequestWithRetry(batchWriteRowUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	for _, table := range resp.Tables {
		index := int32(0)
		for _, row := range table.Rows {
			rowResult := &RowResult{TableName: *table.TableName, IsSucceed: *row.IsOk, ConsumedCapacityUnit: &ConsumedCapacityUnit{}, Index: index}
			index++
			if *row.IsOk == false {
				rowResult.Error = Error{Code: *row.Error.Code, Message: *row.Error.Message}
			} else {
				rowResult.ConsumedCapacityUnit.Read = *row.Consumed.CapacityUnit.Read
				rowResult.ConsumedCapacityUnit.Write = *row.Consumed.CapacityUnit.Write
			} /*else {
				rows, err := readRowsWithHeader(bytes.NewReader(row.Row))
				if err != nil {
					return nil, err
				}

				for _, pk := range (rows[0].primaryKey) {
					pkColumn := &PrimaryKeyColumn{ColumnName: string(pk.cellName), Value: pk.cellValue.Value}
					rowResult.PrimaryKey.PrimaryKeys = append(rowResult.PrimaryKey.PrimaryKeys, pkColumn)
				}

				for _, cell := range (rows[0].cells) {
					dataColumn := &DataColumn{ColumnName: string(cell.cellName), Value: cell.cellValue.Value}
					rowResult.Columns = append(rowResult.Columns, dataColumn)
				}

				rowResult.ConsumedCapacityUnit.Read = *row.Consumed.CapacityUnit.Read
				rowResult.ConsumedCapacityUnit.Write = *row.Consumed.CapacityUnit.Write
			}*/

			response.TableToRowsResult[*table.TableName] = append(response.TableToRowsResult[*table.TableName], *rowResult)
		}
	}
	return response, nil
}

// Get Range
// @param GetRangeRequest
func (tableStoreClient *TableStoreClient) GetRange(request *GetRangeRequest) (*GetRangeResponse, error) {
	req := new(otsprotocol.GetRangeRequest)
	req.TableName = proto.String(request.RangeRowQueryCriteria.TableName)
	req.Direction = request.RangeRowQueryCriteria.Direction.ToDirection().Enum()

	if request.RangeRowQueryCriteria.MaxVersion != 0 {
		req.MaxVersions = proto.Int32(request.RangeRowQueryCriteria.MaxVersion)
	}

	if request.RangeRowQueryCriteria.TransactionId != nil {
		req.TransactionId = request.RangeRowQueryCriteria.TransactionId
	}

	if request.RangeRowQueryCriteria.TimeRange != nil {
		if request.RangeRowQueryCriteria.TimeRange.Specific != 0 {
			req.TimeRange = &otsprotocol.TimeRange{SpecificTime: proto.Int64(request.RangeRowQueryCriteria.TimeRange.Specific)}
		} else {
			req.TimeRange = &otsprotocol.TimeRange{StartTime: proto.Int64(request.RangeRowQueryCriteria.TimeRange.Start), EndTime: proto.Int64(request.RangeRowQueryCriteria.TimeRange.End)}
		}
	} else if request.RangeRowQueryCriteria.MaxVersion == 0 {
		return nil, errInvalidInput
	}

	if request.RangeRowQueryCriteria.Limit != 0 {
		req.Limit = proto.Int32(request.RangeRowQueryCriteria.Limit)
	}

	if (request.RangeRowQueryCriteria.ColumnsToGet != nil) && len(request.RangeRowQueryCriteria.ColumnsToGet) > 0 {
		req.ColumnsToGet = request.RangeRowQueryCriteria.ColumnsToGet
	}

	if request.RangeRowQueryCriteria.Filter != nil {
		req.Filter = request.RangeRowQueryCriteria.Filter.Serialize()
	}

	if request.RangeRowQueryCriteria.StartColumn != nil {
		req.StartColumn = request.RangeRowQueryCriteria.StartColumn
	}

	if request.RangeRowQueryCriteria.EndColumn != nil {
		req.EndColumn = request.RangeRowQueryCriteria.EndColumn
	}

	req.InclusiveStartPrimaryKey = request.RangeRowQueryCriteria.StartPrimaryKey.Build(false)
	req.ExclusiveEndPrimaryKey = request.RangeRowQueryCriteria.EndPrimaryKey.Build(false)

	resp := new(otsprotocol.GetRangeResponse)
	response := &GetRangeResponse{ConsumedCapacityUnit: &ConsumedCapacityUnit{}}
	if err := tableStoreClient.doRequestWithRetry(getRangeUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	response.ConsumedCapacityUnit.Read = *resp.Consumed.CapacityUnit.Read
	response.ConsumedCapacityUnit.Write = *resp.Consumed.CapacityUnit.Write
	if len(resp.NextStartPrimaryKey) != 0 {
		currentRows, err := readRowsWithHeader(bytes.NewReader(resp.NextStartPrimaryKey))
		if err != nil {
			return nil, err
		}

		response.NextStartPrimaryKey = &PrimaryKey{}
		for _, pk := range currentRows[0].primaryKey {
			pkColumn := &PrimaryKeyColumn{ColumnName: string(pk.cellName), Value: pk.cellValue.Value}
			response.NextStartPrimaryKey.PrimaryKeys = append(response.NextStartPrimaryKey.PrimaryKeys, pkColumn)
		}
	}

	if len(resp.Rows) == 0 {
		return response, nil
	}

	rows, err := readRowsWithHeader(bytes.NewReader(resp.Rows))
	if err != nil {
		return response, err
	}

	for _, row := range rows {
		currentRow := &Row{}
		currentpk := new(PrimaryKey)
		for _, pk := range row.primaryKey {
			pkColumn := &PrimaryKeyColumn{ColumnName: string(pk.cellName), Value: pk.cellValue.Value}
			currentpk.PrimaryKeys = append(currentpk.PrimaryKeys, pkColumn)
		}

		currentRow.PrimaryKey = currentpk

		for _, cell := range row.cells {
			dataColumn := &AttributeColumn{ColumnName: string(cell.cellName), Value: cell.cellValue.Value, Timestamp: cell.cellTimestamp}
			currentRow.Columns = append(currentRow.Columns, dataColumn)
		}

		response.Rows = append(response.Rows, currentRow)
	}

	return response, nil

}

func (client *TableStoreClient) ListStream(req *ListStreamRequest) (*ListStreamResponse, error) {
	pbReq := &otsprotocol.ListStreamRequest{}
	pbReq.TableName = req.TableName

	pbResp := otsprotocol.ListStreamResponse{}
	resp := ListStreamResponse{}
	if err := client.doRequestWithRetry(listStreamUri, pbReq, &pbResp, &resp.ResponseInfo); err != nil {
		return nil, err
	}

	streams := make([]Stream, len(pbResp.Streams))
	for i, pbStream := range pbResp.Streams {
		streams[i] = Stream{
			Id:           (*StreamId)(pbStream.StreamId),
			TableName:    pbStream.TableName,
			CreationTime: *pbStream.CreationTime}
	}
	resp.Streams = streams[:]
	return &resp, nil
}

func (client *TableStoreClient) DescribeStream(req *DescribeStreamRequest) (*DescribeStreamResponse, error) {
	pbReq := &otsprotocol.DescribeStreamRequest{}
	{
		pbReq.StreamId = (*string)(req.StreamId)
		pbReq.InclusiveStartShardId = (*string)(req.InclusiveStartShardId)
		pbReq.ShardLimit = req.ShardLimit
	}
	pbResp := otsprotocol.DescribeStreamResponse{}
	resp := DescribeStreamResponse{}
	if err := client.doRequestWithRetry(describeStreamUri, pbReq, &pbResp, &resp.ResponseInfo); err != nil {
		return nil, err
	}

	resp.StreamId = (*StreamId)(pbResp.StreamId)
	resp.ExpirationTime = *pbResp.ExpirationTime
	resp.TableName = pbResp.TableName
	resp.CreationTime = *pbResp.CreationTime
	Assert(pbResp.StreamStatus != nil, "StreamStatus in DescribeStreamResponse is required.")
	switch *pbResp.StreamStatus {
	case otsprotocol.StreamStatus_STREAM_ENABLING:
		resp.Status = SS_Enabling
	case otsprotocol.StreamStatus_STREAM_ACTIVE:
		resp.Status = SS_Active
	}
	resp.NextShardId = (*ShardId)(pbResp.NextShardId)
	shards := make([]*StreamShard, len(pbResp.Shards))
	for i, pbShard := range pbResp.Shards {
		shards[i] = &StreamShard{
			SelfShard:   (*ShardId)(pbShard.ShardId),
			FatherShard: (*ShardId)(pbShard.ParentId),
			MotherShard: (*ShardId)(pbShard.ParentSiblingId)}
	}
	resp.Shards = shards[:]
	return &resp, nil
}

func (client *TableStoreClient) GetShardIterator(req *GetShardIteratorRequest) (*GetShardIteratorResponse, error) {
	pbReq := &otsprotocol.GetShardIteratorRequest{
		StreamId: (*string)(req.StreamId),
		ShardId:  (*string)(req.ShardId)}

	if req.Timestamp != nil {
		pbReq.Timestamp = req.Timestamp
	}

	if req.Token != nil {
		pbReq.Token = req.Token
	}

	pbResp := otsprotocol.GetShardIteratorResponse{}
	resp := GetShardIteratorResponse{}
	if err := client.doRequestWithRetry(getShardIteratorUri, pbReq, &pbResp, &resp.ResponseInfo); err != nil {
		return nil, err
	}

	resp.ShardIterator = (*ShardIterator)(pbResp.ShardIterator)
	resp.Token = pbResp.NextToken
	return &resp, nil
}

func (client TableStoreClient) GetStreamRecord(req *GetStreamRecordRequest) (*GetStreamRecordResponse, error) {
	pbReq := &otsprotocol.GetStreamRecordRequest{
		ShardIterator: (*string)(req.ShardIterator)}
	if req.Limit != nil {
		pbReq.Limit = req.Limit
	}

	pbResp := otsprotocol.GetStreamRecordResponse{}
	resp := GetStreamRecordResponse{}
	if err := client.doRequestWithRetry(getStreamRecordUri, pbReq, &pbResp, &resp.ResponseInfo); err != nil {
		return nil, err
	}

	if pbResp.NextShardIterator != nil {
		resp.NextShardIterator = (*ShardIterator)(pbResp.NextShardIterator)
	}
	records := make([]*StreamRecord, len(pbResp.StreamRecords))
	for i, pbRecord := range pbResp.StreamRecords {
		record := StreamRecord{}
		records[i] = &record

		switch *pbRecord.ActionType {
		case otsprotocol.ActionType_PUT_ROW:
			record.Type = AT_Put
		case otsprotocol.ActionType_UPDATE_ROW:
			record.Type = AT_Update
		case otsprotocol.ActionType_DELETE_ROW:
			record.Type = AT_Delete
		}

		plainRows, err := readRowsWithHeader(bytes.NewReader(pbRecord.Record))
		if err != nil {
			return nil, err
		}
		Assert(len(plainRows) == 1,
			"There must be exactly one row in a StreamRecord.")
		plainRow := plainRows[0]
		pkey := PrimaryKey{}
		record.PrimaryKey = &pkey
		pkey.PrimaryKeys = make([]*PrimaryKeyColumn, len(plainRow.primaryKey))
		for i, pk := range plainRow.primaryKey {
			pkc := PrimaryKeyColumn{
				ColumnName: string(pk.cellName),
				Value:      pk.cellValue.Value}
			pkey.PrimaryKeys[i] = &pkc
		}
		Assert(plainRow.extension != nil,
			"extension in a stream record is required.")
		record.Info = plainRow.extension
		record.Columns = make([]*RecordColumn, len(plainRow.cells))
		for i, plainCell := range plainRow.cells {
			cell := RecordColumn{}
			record.Columns[i] = &cell

			name := string(plainCell.cellName)
			cell.Name = &name
			if plainCell.cellValue != nil {
				cell.Type = RCT_Put
			} else {
				if plainCell.cellTimestamp > 0 {
					cell.Type = RCT_DeleteOneVersion
				} else {
					cell.Type = RCT_DeleteAllVersions
				}
			}
			switch cell.Type {
			case RCT_Put:
				cell.Value = plainCell.cellValue.Value
				fallthrough
			case RCT_DeleteOneVersion:
				cell.Timestamp = &plainCell.cellTimestamp
			case RCT_DeleteAllVersions:
				break
			}
		}
	}
	resp.Records = records
	return &resp, nil
}

func (client TableStoreClient) ComputeSplitPointsBySize(req *ComputeSplitPointsBySizeRequest) (*ComputeSplitPointsBySizeResponse, error) {
	pbReq := &otsprotocol.ComputeSplitPointsBySizeRequest{
		TableName: &(req.TableName),
		SplitSize: &(req.SplitSize),
	}

	pbResp := otsprotocol.ComputeSplitPointsBySizeResponse{}
	resp := ComputeSplitPointsBySizeResponse{}
	if err := client.doRequestWithRetry(computeSplitPointsBySizeRequestUri, pbReq, &pbResp, &resp.ResponseInfo); err != nil {
		return nil, err
	}

	fmt.Println(len(pbResp.SplitPoints))
	fmt.Println(len(pbResp.Locations))

	beginPk := &PrimaryKey{}
	endPk := &PrimaryKey{}
	for _, pkSchema := range pbResp.Schema {
		beginPk.AddPrimaryKeyColumnWithMinValue(*pkSchema.Name)
		endPk.AddPrimaryKeyColumnWithMaxValue(*pkSchema.Name)
	}
	lastPk := beginPk
	nowPk := endPk

	for _, pbRecord := range pbResp.SplitPoints {
		plainRows, err := readRowsWithHeader(bytes.NewReader(pbRecord))
		if err != nil {
			return nil, err
		}

		nowPk = &PrimaryKey{}
		for _, pk := range plainRows[0].primaryKey {
			nowPk.AddPrimaryKeyColumn(string(pk.cellName), pk.cellValue.Value)
		}

		if len(pbResp.Schema) > 1 {
			for i := 1; i < len(pbResp.Schema); i++ {
				nowPk.AddPrimaryKeyColumnWithMinValue(*pbResp.Schema[i].Name)
			}
		}

		newSplit := &Split{LowerBound: lastPk, UpperBound: nowPk}
		resp.Splits = append(resp.Splits, newSplit)
		lastPk = nowPk

	}

	newSplit := &Split{LowerBound: lastPk, UpperBound: endPk}
	resp.Splits = append(resp.Splits, newSplit)

	index := 0
	for _, pbLocation := range pbResp.Locations {
		count := *pbLocation.Repeat
		value := *pbLocation.Location

		for i := int64(0); i < count; i++ {
			resp.Splits[index].Location = value
			index++
		}
	}
	return &resp, nil
}

func (client *TableStoreClient) StartLocalTransaction(request *StartLocalTransactionRequest) (*StartLocalTransactionResponse, error) {
	req := new(otsprotocol.StartLocalTransactionRequest)
	resp := new(otsprotocol.StartLocalTransactionResponse)

	req.TableName = proto.String(request.TableName)
	req.Key = request.PrimaryKey.Build(false)

	response := &StartLocalTransactionResponse{}
	if err := client.doRequestWithRetry(createlocaltransactionuri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	response.TransactionId = resp.TransactionId
	return response, nil
}

func (client *TableStoreClient) CommitTransaction(request *CommitTransactionRequest) (*CommitTransactionResponse, error) {
	req := new(otsprotocol.CommitTransactionRequest)
	resp := new(otsprotocol.CommitTransactionResponse)

	req.TransactionId = request.TransactionId

	response := &CommitTransactionResponse{}
	if err := client.doRequestWithRetry(committransactionuri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	return response, nil
}

func (client *TableStoreClient) AbortTransaction(request *AbortTransactionRequest) (*AbortTransactionResponse, error) {
	req := new(otsprotocol.AbortTransactionRequest)
	resp := new(otsprotocol.AbortTransactionResponse)

	req.TransactionId = request.TransactionId

	response := &AbortTransactionResponse{}
	if err := client.doRequestWithRetry(aborttransactionuri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}

	return response, nil
}