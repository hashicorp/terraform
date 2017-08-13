// Package implements OCCM Audit API
package audit

import (
  "encoding/json"

  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Audit group summary object
type AuditGroupSummary struct {
  Id                      int32  `json:"_id"`
  PublicId                string `json:"id"`
  RequestId               string `json:"requestId"`
  StartDate               int64  `json:"startDate"`
  EndDate                 int64  `json:"endDate"`
  ActionName              string `json:"actionName"`
  Status                  string `json:"status"`
  TenantName              string `json:"tenantName"`
  WorkingEnvironmentName  string `json:"workingEnvironmentName"`
  ActionParameters        string `json:"actionParameters"`
  Records  []AuditGroupSummaryRecord `json:"records"`
  ErrorMessage            string `json:"errorMessage"`
  Version                 string `json:"version"`
  // ParentId  string `json:"parentId"`
  ContainsFailedRecords   bool `json:"containsFailedRecords"`
}

// Audit group summary record object
type AuditGroupSummaryRecord struct {
  Id             string `json:"id"`
  Date           int64  `json:"date"`
  ActionName     string `json:"actionName"`
  Status         string `json:"status"`
  Parameters     string `json:"parameters"`
  ErrorMessage   string `json:"errorMessage"`
  Count          int32  `json:"count"`
}

func ListFromJSON(data []byte) ([]AuditGroupSummary, error) {
  var result []AuditGroupSummary
  err := json.Unmarshal(data, &result)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return result, nil
}
