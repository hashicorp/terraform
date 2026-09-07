// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

// This file is copied from terraform-provider-azurerm: internal/provider/helpers.go

package azure

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/backendbase"
)

// logEntry avoids log entries showing up in test output
func logEntry(f string, v ...interface{}) {
	if os.Getenv("TF_LOG") == "" {
		return
	}

	if os.Getenv("TF_ACC") != "" {
		return
	}

	log.Printf(f, v...)
}

func decodeCertificate(clientCertificate string) ([]byte, error) {
	var pfx []byte
	if clientCertificate != "" {
		out := make([]byte, base64.StdEncoding.DecodedLen(len(clientCertificate)))
		n, err := base64.StdEncoding.Decode(out, []byte(clientCertificate))
		if err != nil {
			return pfx, fmt.Errorf("could not decode client certificate data: %v", err)
		}
		pfx = out[:n]
	}
	return pfx, nil
}

func getOidcToken(d *backendbase.SDKLikeData) (*string, error) {
	idToken := strings.TrimSpace(d.String("oidc_token"))
	tokenFilePath := d.String("oidc_token_file_path")

	if environmentVariableSet("ARM_OIDC_TOKEN_BACKEND", "ARM_OIDC_TOKEN_FILE_PATH_BACKEND") {
		idToken = strings.TrimSpace(backendbase.SDKLikeEnvDefault(idToken, "ARM_OIDC_TOKEN_BACKEND"))
		tokenFilePath = backendbase.SDKLikeEnvDefault(tokenFilePath, "ARM_OIDC_TOKEN_FILE_PATH_BACKEND")
	} else {
		idToken = strings.TrimSpace(backendbase.SDKLikeEnvDefault(idToken, "ARM_OIDC_TOKEN"))
		tokenFilePath = backendbase.SDKLikeEnvDefault(tokenFilePath, "ARM_OIDC_TOKEN_FILE_PATH")
	}

	if path := tokenFilePath; path != "" {
		fileTokenRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading OIDC Token from file %q: %v", path, err)
		}

		fileToken := strings.TrimSpace(string(fileTokenRaw))

		if idToken != "" && idToken != fileToken {
			return nil, fmt.Errorf("mismatch between supplied OIDC token and supplied OIDC token file contents - please either remove one or ensure they match")
		}

		idToken = fileToken
	}

	if d.Bool("use_aks_workload_identity") && os.Getenv("AZURE_FEDERATED_TOKEN_FILE") != "" {
		path := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
		fileTokenRaw, err := os.ReadFile(os.Getenv("AZURE_FEDERATED_TOKEN_FILE"))

		if err != nil {
			return nil, fmt.Errorf("reading OIDC Token from file %q provided by AKS Workload Identity: %v", path, err)
		}

		fileToken := strings.TrimSpace(string(fileTokenRaw))

		if idToken != "" && idToken != fileToken {
			return nil, fmt.Errorf("mismatch between supplied OIDC token and OIDC token file contents provided by AKS Workload Identity - please either remove one, ensure they match, or disable use_aks_workload_identity")
		}

		idToken = fileToken
	}

	return &idToken, nil
}

func getClientId(d *backendbase.SDKLikeData) (*string, error) {
	clientId := strings.TrimSpace(d.String("client_id"))
	clientIdFilePath := d.String("client_id_file_path")

	if environmentVariableSet("ARM_CLIENT_ID_BACKEND", "ARM_CLIENT_ID_FILE_PATH_BACKEND") {
		clientId = strings.TrimSpace(backendbase.SDKLikeEnvDefault(clientId, "ARM_CLIENT_ID_BACKEND"))
		clientIdFilePath = backendbase.SDKLikeEnvDefault(clientIdFilePath, "ARM_CLIENT_ID_FILE_PATH_BACKEND")
	} else {
		clientId = strings.TrimSpace(backendbase.SDKLikeEnvDefault(clientId, "ARM_CLIENT_ID"))
		clientIdFilePath = backendbase.SDKLikeEnvDefault(clientIdFilePath, "ARM_CLIENT_ID_FILE_PATH")
	}

	if path := clientIdFilePath; path != "" {
		fileClientIdRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading Client ID from file %q: %v", path, err)
		}

		fileClientId := strings.TrimSpace(string(fileClientIdRaw))

		if clientId != "" && clientId != fileClientId {
			return nil, fmt.Errorf("mismatch between supplied Client ID and supplied Client ID file contents - please either remove one or ensure they match")
		}

		clientId = fileClientId
	}

	if d.Bool("use_aks_workload_identity") && os.Getenv("AZURE_CLIENT_ID") != "" {
		aksClientId := os.Getenv("AZURE_CLIENT_ID")
		if clientId != "" && clientId != aksClientId {
			return nil, fmt.Errorf("mismatch between supplied Client ID and that provided by AKS Workload Identity - please remove, ensure they match, or disable use_aks_workload_identity")
		}
		clientId = aksClientId
	}

	return &clientId, nil
}

func getClientSecret(d *backendbase.SDKLikeData) (*string, error) {
	clientSecret := strings.TrimSpace(d.String("client_secret"))

	if path := d.String("client_secret_file_path"); path != "" {
		fileSecretRaw, err := os.ReadFile(path)

		if err != nil {
			return nil, fmt.Errorf("reading Client Secret from file %q: %v", path, err)
		}

		fileSecret := strings.TrimSpace(string(fileSecretRaw))

		if clientSecret != "" && clientSecret != fileSecret {
			return nil, fmt.Errorf("mismatch between supplied Client Secret and supplied Client Secret file contents - please either remove one or ensure they match")
		}

		clientSecret = fileSecret
	}

	return &clientSecret, nil
}

func getTenantId(d *backendbase.SDKLikeData) (*string, error) {
	tenantId := strings.TrimSpace(d.String("tenant_id"))

	if d.Bool("use_aks_workload_identity") && os.Getenv("AZURE_TENANT_ID") != "" {
		aksTenantId := os.Getenv("AZURE_TENANT_ID")
		if tenantId != "" && tenantId != aksTenantId {
			return nil, fmt.Errorf("mismatch between supplied Tenant ID and that provided by AKS Workload Identity - please remove, ensure they match, or disable use_aks_workload_identity")
		}
		tenantId = aksTenantId
	}

	return &tenantId, nil
}

func getOidcRequestURL(d *backendbase.SDKLikeData, adoPipelineServiceConnectionID string) string {
	requestURL := backendbase.SDKLikeEnvDefault(
		d.String("oidc_request_url"),
		"ARM_OIDC_REQUEST_URL_BACKEND",
		"ARM_OIDC_REQUEST_URL",
		"ACTIONS_ID_TOKEN_REQUEST_URL",
	)
	if requestURL == "" && adoPipelineServiceConnectionID != "" {
		requestURL = os.Getenv("SYSTEM_OIDCREQUESTURI")
	}
	return requestURL
}

func getOidcRequestToken(d *backendbase.SDKLikeData, adoPipelineServiceConnectionID string) string {
	requestToken := backendbase.SDKLikeEnvDefault(
		d.String("oidc_request_token"),
		"ARM_OIDC_REQUEST_TOKEN_BACKEND",
		"ARM_OIDC_REQUEST_TOKEN",
		"ACTIONS_ID_TOKEN_REQUEST_TOKEN",
	)
	if requestToken == "" && adoPipelineServiceConnectionID != "" {
		requestToken = os.Getenv("SYSTEM_ACCESSTOKEN")
	}
	return requestToken
}

func getADOPipelineServiceConnectionID(d *backendbase.SDKLikeData) string {
	if serviceConnectionID := d.String("ado_pipeline_service_connection_id"); serviceConnectionID != "" {
		return serviceConnectionID
	}

	if backendOIDCIdentityEnvironmentConfigured() {
		return ""
	}

	return backendbase.SDKLikeEnvDefault(
		"",
		"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID",
		"ARM_OIDC_AZURE_SERVICE_CONNECTION_ID",
		"AZURESUBSCRIPTION_SERVICE_CONNECTION_ID",
	)
}

func backendOIDCIdentityEnvironmentConfigured() bool {
	return environmentVariableSet(
		"ARM_TENANT_ID_BACKEND",
		"ARM_CLIENT_ID_BACKEND",
		"ARM_CLIENT_ID_FILE_PATH_BACKEND",
		"ARM_OIDC_REQUEST_TOKEN_BACKEND",
		"ARM_OIDC_REQUEST_URL_BACKEND",
		"ARM_OIDC_TOKEN_BACKEND",
		"ARM_OIDC_TOKEN_FILE_PATH_BACKEND",
	)
}

func environmentVariableSet(names ...string) bool {
	for _, name := range names {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}
