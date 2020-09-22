package tablestore

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"
	"github.com/golang/protobuf/proto"
)

func (tableStoreClient *TableStoreClient) CreateSearchIndex(request *CreateSearchIndexRequest) (*CreateSearchIndexResponse, error) {
	req := new(otsprotocol.CreateSearchIndexRequest)
	req.TableName = proto.String(request.TableName)
	req.IndexName = proto.String(request.IndexName)
	var err error
	req.Schema, err = convertToPbSchema(request.IndexSchema)
	if err != nil {
		return nil, err
	}
	resp := new(otsprotocol.CreateSearchIndexRequest)
	response := &CreateSearchIndexResponse{}
	if err := tableStoreClient.doRequestWithRetry(createSearchIndexUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}
	return response, nil
}

func (tableStoreClient *TableStoreClient) DeleteSearchIndex(request *DeleteSearchIndexRequest) (*DeleteSearchIndexResponse, error) {
	req := new(otsprotocol.DeleteSearchIndexRequest)
	req.TableName = proto.String(request.TableName)
	req.IndexName = proto.String(request.IndexName)

	resp := new(otsprotocol.DeleteSearchIndexResponse)
	response := &DeleteSearchIndexResponse{}
	if err := tableStoreClient.doRequestWithRetry(deleteSearchIndexUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}
	return response, nil
}

func (tableStoreClient *TableStoreClient) ListSearchIndex(request *ListSearchIndexRequest) (*ListSearchIndexResponse, error) {
	req := new(otsprotocol.ListSearchIndexRequest)
	req.TableName = proto.String(request.TableName)

	resp := new(otsprotocol.ListSearchIndexResponse)
	response := &ListSearchIndexResponse{}
	if err := tableStoreClient.doRequestWithRetry(listSearchIndexUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}
	indexs := make([]*IndexInfo, 0)
	for _, info := range resp.Indices {
		indexs = append(indexs, &IndexInfo{
			TableName: *info.TableName,
			IndexName: *info.IndexName,
		})
	}
	response.IndexInfo = indexs
	return response, nil
}

func (tableStoreClient *TableStoreClient) DescribeSearchIndex(request *DescribeSearchIndexRequest) (*DescribeSearchIndexResponse, error) {
	req := new(otsprotocol.DescribeSearchIndexRequest)
	req.TableName = proto.String(request.TableName)
	req.IndexName = proto.String(request.IndexName)

	resp := new(otsprotocol.DescribeSearchIndexResponse)
	response := &DescribeSearchIndexResponse{}
	if err := tableStoreClient.doRequestWithRetry(describeSearchIndexUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}
	schema, err := parseFromPbSchema(resp.Schema)
	if err != nil {
		return nil, err
	}
	response.Schema = schema
	if resp.SyncStat != nil {
		response.SyncStat = &SyncStat{
			CurrentSyncTimestamp: resp.SyncStat.CurrentSyncTimestamp,
		}
		syncPhase := resp.SyncStat.SyncPhase
		if syncPhase == nil {
			return nil, errors.New("missing [SyncPhase] in DescribeSearchIndexResponse")
		} else if *syncPhase == otsprotocol.SyncPhase_FULL {
			response.SyncStat.SyncPhase = SyncPhase_FULL
		} else if *syncPhase == otsprotocol.SyncPhase_INCR {
			response.SyncStat.SyncPhase = SyncPhase_INCR
		} else {
			return nil, errors.New(fmt.Sprintf("unknown SyncPhase: %v", syncPhase))
		}
	}
	return response, nil
}

func (tableStoreClient *TableStoreClient) Search(request *SearchRequest) (*SearchResponse, error) {
	req, err := request.ProtoBuffer()
	if err != nil {
		return nil, err
	}
	resp := new(otsprotocol.SearchResponse)
	response := &SearchResponse{}
	if err := tableStoreClient.doRequestWithRetry(searchUri, req, resp, &response.ResponseInfo); err != nil {
		return nil, err
	}
	response.TotalCount = *resp.TotalHits

	rows := make([]*PlainBufferRow, 0)
	for _, buf := range resp.Rows {
		row, err := readRowsWithHeader(bytes.NewReader(buf))
		if err != nil {
			return nil, err
		}
		rows = append(rows, row[0])
	}

	for _, row := range rows {
		currentRow := &Row{}
		currentPk := new(PrimaryKey)
		for _, pk := range row.primaryKey {
			pkColumn := &PrimaryKeyColumn{ColumnName: string(pk.cellName), Value: pk.cellValue.Value}
			currentPk.PrimaryKeys = append(currentPk.PrimaryKeys, pkColumn)
		}
		currentRow.PrimaryKey = currentPk
		for _, cell := range row.cells {
			dataColumn := &AttributeColumn{ColumnName: string(cell.cellName), Value: cell.cellValue.Value, Timestamp: cell.cellTimestamp}
			currentRow.Columns = append(currentRow.Columns, dataColumn)
		}
		response.Rows = append(response.Rows, currentRow)
	}

	response.IsAllSuccess = *resp.IsAllSucceeded
	if resp.NextToken != nil && len(resp.NextToken) > 0 {
		response.NextToken = resp.NextToken
	}
	return response, nil
}
