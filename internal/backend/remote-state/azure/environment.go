// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"os"
	"strings"

	"github.com/hashicorp/terraform/internal/backend/backendbase"
)

func resolveClientIDInputs(d *backendbase.SDKLikeData) (string, string) {
	clientID := strings.TrimSpace(d.String("client_id"))
	clientIDFilePath := d.String("client_id_file_path")

	if environmentVariableSet("ARM_CLIENT_ID_BACKEND", "ARM_CLIENT_ID_FILE_PATH_BACKEND") {
		clientID = strings.TrimSpace(backendbase.SDKLikeEnvDefault(clientID, "ARM_CLIENT_ID_BACKEND"))
		clientIDFilePath = backendbase.SDKLikeEnvDefault(clientIDFilePath, "ARM_CLIENT_ID_FILE_PATH_BACKEND")
	} else {
		clientID = strings.TrimSpace(backendbase.SDKLikeEnvDefault(clientID, "ARM_CLIENT_ID"))
		clientIDFilePath = backendbase.SDKLikeEnvDefault(clientIDFilePath, "ARM_CLIENT_ID_FILE_PATH")
	}

	return clientID, clientIDFilePath
}

func resolveOidcTokenInputs(d *backendbase.SDKLikeData) (string, string) {
	idToken := strings.TrimSpace(d.String("oidc_token"))
	tokenFilePath := d.String("oidc_token_file_path")

	if environmentVariableSet("ARM_OIDC_TOKEN_BACKEND", "ARM_OIDC_TOKEN_FILE_PATH_BACKEND") {
		idToken = strings.TrimSpace(backendbase.SDKLikeEnvDefault(idToken, "ARM_OIDC_TOKEN_BACKEND"))
		tokenFilePath = backendbase.SDKLikeEnvDefault(tokenFilePath, "ARM_OIDC_TOKEN_FILE_PATH_BACKEND")
	} else {
		idToken = strings.TrimSpace(backendbase.SDKLikeEnvDefault(idToken, "ARM_OIDC_TOKEN"))
		tokenFilePath = backendbase.SDKLikeEnvDefault(tokenFilePath, "ARM_OIDC_TOKEN_FILE_PATH")
	}

	return idToken, tokenFilePath
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
