// Package implements OCCM Audit API
package audit

type AuditAPIProto interface {
	GetAuditSummaries(limit, after int64, workenvId string) ([]AuditGroupSummary, error)
  GetAuditSummary(requestId string) (*AuditGroupSummary, error)
}

var _ AuditAPIProto = (*AuditAPI)(nil)
