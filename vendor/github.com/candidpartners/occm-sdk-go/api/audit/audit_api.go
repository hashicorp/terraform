// Package implements OCCM Audit API
package audit

import (
  "fmt"

  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Tenant API
type AuditAPI struct {
	*client.Client
}

// New creates a new OCCM Audit API client
func New(context *client.Context) (*AuditAPI, error) {
  c, err := client.New(context)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrClientCreationFailed)
  }

	api := &AuditAPI{
		Client: c,
	}

	return api, nil
}

// GetAuditSummaries retrieves a list of audit summaries
func (api *AuditAPI) GetAuditSummaries(limit, after int64, workenvId string) ([]AuditGroupSummary, error) {
  data, _, err := api.Client.Invoke("GET", "/audit",
    map[string]string{
      "limit": fmt.Sprintf("%v", limit),
      "after": fmt.Sprintf("%v", after),
      "workingEnvironmentId": workenvId,
    },
    nil,
  )
  if err != nil {
		return nil, errors.Wrap(err, client.ErrInvalidRequest)
	}

  result, err := ListFromJSON(data);
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return result, nil
}

// GetAuditSummary retrieves an audit summary
func (api *AuditAPI) GetAuditSummary(requestId string) (*AuditGroupSummary, error) {
  data, _, err := api.Client.Invoke("GET",
    fmt.Sprintf("/audit/%v", requestId),
    nil,
    nil,
  )
  if err != nil {
		return nil, errors.Wrap(err, client.ErrInvalidRequest)
	}

  result, err := ListFromJSON(data);
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  if len(result) == 0 {
    return nil, errors.Wrap(err, client.ErrInvalidRequest)
  }

  return &result[0], nil
}
