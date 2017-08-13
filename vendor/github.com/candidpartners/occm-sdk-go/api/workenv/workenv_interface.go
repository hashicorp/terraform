// Package implements OCCM Working Environments API
package workenv

type WorkingEnvironmentAPIProto interface {
	GetWorkingEnvironments() (*WorkingEnvironments, error)
}

var _ WorkingEnvironmentAPIProto = (*WorkingEnvironmentAPI)(nil)
