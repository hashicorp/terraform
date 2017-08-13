// Package implements OCCM Tenant API
package tenant

type TenantAPIProto interface {
	GetTenants() ([]Tenant, error)
}

var _ TenantAPIProto = (*TenantAPI)(nil)
