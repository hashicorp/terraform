// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package elasticsearch

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	uuid "github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// RemoteClient is a remote client that stores data in Elasticsearch
type RemoteClient struct {
	Client      *elasticsearch.Client
	Index       string
	LockEnabled bool
	Workspace   string

	info *statemgr.LockInfo
}

// stateDocument represents the Elasticsearch document structure for state storage
type stateDocument struct {
	Data      string    `json:"data"`
	MD5       string    `json:"md5"`
	Workspace string    `json:"workspace"`
	UpdatedAt time.Time `json:"updated_at"`
}

// lockDocument represents the Elasticsearch document structure for locking
type lockDocument struct {
	LockInfo  string    `json:"lock_info"`
	CreatedAt time.Time `json:"created_at"`
	Workspace string    `json:"workspace"`
}

// documentID returns the document ID for the current workspace
func (c *RemoteClient) documentID() string {
	return fmt.Sprintf("state-%s", c.Workspace)
}

// lockDocumentID returns the lock document ID for the current workspace
func (c *RemoteClient) lockDocumentID() string {
	return fmt.Sprintf("lock-%s", c.Workspace)
}

func (c *RemoteClient) Get() (*remote.Payload, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	req := esapi.GetRequest{
		Index:      c.Index,
		DocumentID: c.documentID(),
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return nil, diags.Append(fmt.Errorf("failed to GET state: %w", err))
	}
	defer res.Body.Close()

	// Handle not found - return nil payload (no state yet)
	if res.StatusCode == 404 {
		return nil, diags
	}

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, diags.Append(fmt.Errorf("failed to GET state: status=%d, body=%s", res.StatusCode, string(body)))
	}

	// Parse response
	var esResponse struct {
		Found  bool          `json:"found"`
		Source stateDocument `json:"_source"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, diags.Append(fmt.Errorf("failed to decode response: %w", err))
	}

	if !esResponse.Found {
		return nil, diags
	}

	// Decode state data
	data := []byte(esResponse.Source.Data)
	if len(data) == 0 {
		return nil, diags
	}

	// Calculate MD5
	hash := md5.Sum(data)

	return &remote.Payload{
		Data: data,
		MD5:  hash[:],
	}, diags
}

func (c *RemoteClient) Put(data []byte) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Calculate MD5
	hash := md5.Sum(data)
	md5Str := fmt.Sprintf("%x", hash)

	// Create document
	doc := stateDocument{
		Data:      string(data),
		MD5:       md5Str,
		Workspace: c.Workspace,
		UpdatedAt: time.Now(),
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return diags.Append(fmt.Errorf("failed to marshal document: %w", err))
	}

	req := esapi.IndexRequest{
		Index:      c.Index,
		DocumentID: c.documentID(),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return diags.Append(fmt.Errorf("failed to PUT state: %w", err))
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return diags.Append(fmt.Errorf("failed to PUT state: status=%d, body=%s", res.StatusCode, string(body)))
	}

	return diags
}

func (c *RemoteClient) Delete() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	req := esapi.DeleteRequest{
		Index:      c.Index,
		DocumentID: c.documentID(),
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return diags.Append(fmt.Errorf("failed to DELETE state: %w", err))
	}
	defer res.Body.Close()

	// Accept both OK and Not Found as success
	if res.IsError() && res.StatusCode != 404 {
		body, _ := io.ReadAll(res.Body)
		return diags.Append(fmt.Errorf("failed to DELETE state: status=%d, body=%s", res.StatusCode, string(body)))
	}

	return diags
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	if !c.LockEnabled {
		return "", nil
	}

	if info.ID == "" {
		lockID, err := uuid.GenerateUUID()
		if err != nil {
			return "", err
		}
		info.ID = lockID
	}

	lockInfo := info.Marshal()

	// Create lock document
	doc := lockDocument{
		LockInfo:  string(lockInfo),
		CreatedAt: time.Now(),
		Workspace: c.Workspace,
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("failed to marshal lock document: %w", err)
	}

	// Try to create the lock document with op_type=create (only succeeds if doesn't exist)
	req := esapi.CreateRequest{
		Index:      c.Index,
		DocumentID: c.lockDocumentID(),
		Body:       bytes.NewReader(body),
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer res.Body.Close()

	if !res.IsError() {
		// Lock acquired successfully
		c.info = info
		return info.ID, nil
	}

	// Lock already exists (conflict)
	if res.StatusCode == 409 {
		// Try to get existing lock info
		existingInfo, err := c.getExistingLockInfo()
		if err != nil {
			return "", &statemgr.LockError{
				Info: info,
				Err:  fmt.Errorf("state is locked, but failed to read lock info: %w", err),
			}
		}
		return "", &statemgr.LockError{
			Info: existingInfo,
			Err:  fmt.Errorf("state is locked by another process"),
		}
	}

	respBody, _ := io.ReadAll(res.Body)
	return "", fmt.Errorf("unexpected response when acquiring lock: status=%d, body=%s", res.StatusCode, string(respBody))
}

func (c *RemoteClient) Unlock(id string) error {
	if !c.LockEnabled {
		return nil
	}

	req := esapi.DeleteRequest{
		Index:      c.Index,
		DocumentID: c.lockDocumentID(),
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to release lock: status=%d, body=%s", res.StatusCode, string(body))
	}

	c.info = nil
	return nil
}

func (c *RemoteClient) getLockInfo() (*statemgr.LockInfo, error) {
	return c.info, nil
}

// getExistingLockInfo retrieves information about an existing lock from Elasticsearch
func (c *RemoteClient) getExistingLockInfo() (*statemgr.LockInfo, error) {
	req := esapi.GetRequest{
		Index:      c.Index,
		DocumentID: c.lockDocumentID(),
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to get lock info: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("lock document not found")
	}

	var esResponse struct {
		Found  bool         `json:"found"`
		Source lockDocument `json:"_source"`
	}

	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode lock response: %w", err)
	}

	if !esResponse.Found {
		return nil, fmt.Errorf("lock document not found")
	}

	// Parse lock info
	lockInfo := &statemgr.LockInfo{}
	if err := json.Unmarshal([]byte(esResponse.Source.LockInfo), lockInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal lock info: %w", err)
	}

	return lockInfo, nil
}

// Workspaces returns all workspaces stored in Elasticsearch
func (c *RemoteClient) Workspaces() ([]string, error) {
	// Search for all state documents by checking existence of "workspace" field
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"exists": map[string]interface{}{
				"field": "workspace",
			},
		},
		"_source": []string{"workspace"},
		"size":    1000, // adjust if needed
	}

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{c.Index},
		Body:  bytes.NewReader(body),
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to search workspaces: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("failed to search workspaces: status=%d, body=%s", res.StatusCode, string(body))
	}

	var searchResponse struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Workspace string `json:"workspace"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	workspaces := []string{}
	seen := make(map[string]bool)

	for _, hit := range searchResponse.Hits.Hits {
		ws := hit.Source.Workspace
		if ws != "" && !seen[ws] {
			workspaces = append(workspaces, ws)
			seen[ws] = true
		}
	}

	return workspaces, nil
}

// DeleteWorkspace removes a workspace's state
func (c *RemoteClient) DeleteWorkspace() error {
	// Delete state document
	stateDiags := c.Delete()
	if len(stateDiags) > 0 {
		return fmt.Errorf("failed to delete state: %v", stateDiags.Err())
	}

	// Delete lock document if it exists
	req := esapi.DeleteRequest{
		Index:      c.Index,
		DocumentID: c.lockDocumentID(),
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return fmt.Errorf("failed to delete lock: %w", err)
	}
	defer res.Body.Close()

	// Ignore 404 errors for lock deletion
	if res.IsError() && res.StatusCode != 404 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to delete lock: status=%d, body=%s", res.StatusCode, string(body))
	}

	return nil
}

// ensureIndex ensures the Elasticsearch index exists
func (c *RemoteClient) ensureIndex() error {
	// Check if index exists
	req := esapi.IndicesExistsRequest{
		Index: []string{c.Index},
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return fmt.Errorf("failed to check index: %w", err)
	}
	defer res.Body.Close()

	// If index exists, we're done
	if res.StatusCode == 200 {
		return nil
	}

	// Create the index
	indexMapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"data": map[string]interface{}{
					"type":  "text",
					"index": false,
				},
				"md5": map[string]interface{}{
					"type": "keyword",
				},
				"workspace": map[string]interface{}{
					"type": "keyword",
				},
				"updated_at": map[string]interface{}{
					"type": "date",
				},
				"lock_info": map[string]interface{}{
					"type":  "text",
					"index": false,
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}

	body, err := json.Marshal(indexMapping)
	if err != nil {
		return fmt.Errorf("failed to marshal index mapping: %w", err)
	}

	createReq := esapi.IndicesCreateRequest{
		Index: c.Index,
		Body:  bytes.NewReader(body),
	}

	createRes, err := createReq.Do(context.Background(), c.Client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		body, _ := io.ReadAll(createRes.Body)
		return fmt.Errorf("failed to create index: status=%d, body=%s", createRes.StatusCode, string(body))
	}

	return nil
}

// deleteIndex deletes the entire Elasticsearch index (used for testing cleanup)
func (c *RemoteClient) deleteIndex() error {
	req := esapi.IndicesDeleteRequest{
		Index: []string{c.Index},
	}

	res, err := req.Do(context.Background(), c.Client)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	// Accept 200 OK or 404 Not Found as success
	if res.IsError() && res.StatusCode != 404 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to delete index: status=%d, body=%s", res.StatusCode, string(body))
	}

	return nil
}
