package netapp

import (
	"fmt"
	"log"

	"github.com/candidpartners/occm-sdk-go/api/audit"
	"github.com/candidpartners/occm-sdk-go/api/auth"
	"github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/candidpartners/occm-sdk-go/api/tenant"
	"github.com/candidpartners/occm-sdk-go/api/workenv"
	"github.com/candidpartners/occm-sdk-go/api/workenv/awsha"
	"github.com/candidpartners/occm-sdk-go/api/workenv/vsa"
)

type APIs struct {
	*auth.AuthAPI
	*tenant.TenantAPI
	*workenv.WorkingEnvironmentAPI
	*vsa.VSAWorkingEnvironmentAPI
	*awsha.AWSHAWorkingEnvironmentAPI
	*audit.AuditAPI
}

type Config struct {
	Host     string
	Email    string
	Password string
}

func (c *Config) APIs() (*APIs, error) {
	context := &client.Context{
		Host: c.Host,
	}

	authApi, err := auth.New(context)
	if err != nil {
		return nil, fmt.Errorf("Error creating auth API: %s", err)
	}

	tenantApi, err := tenant.New(context)
	if err != nil {
		return nil, fmt.Errorf("Error creating tenant API: %s", err)
	}

	workenvApi, err := workenv.New(context)
	if err != nil {
		return nil, fmt.Errorf("Error creating working environment API: %s", err)
	}

	vsaWorkenvApi, err := vsa.New(context)
	if err != nil {
		return nil, fmt.Errorf("Error creating VSA working environment API: %s", err)
	}

	awsHaWorkenvApi, err := awsha.New(context)
	if err != nil {
		return nil, fmt.Errorf("Error creating AWS HA working environment API: %s", err)
	}

	auditApi, err := audit.New(context)
	if err != nil {
		return nil, fmt.Errorf("Error creating audit API: %s", err)
	}

	apis := &APIs{
		AuthAPI:                    authApi,
		TenantAPI:                  tenantApi,
		WorkingEnvironmentAPI:      workenvApi,
		VSAWorkingEnvironmentAPI:   vsaWorkenvApi,
		AWSHAWorkingEnvironmentAPI: awsHaWorkenvApi,
		AuditAPI:                   auditApi,
	}

	log.Printf("[INFO] NetApp Client configured for user: %s", c.Email)
	return apis, nil
}
