package net

import (
	"encoding/json"
	"time"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
)

type uaaErrorResponse struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

var uaaErrorHandler = func(statusCode int, body []byte) error {
	response := uaaErrorResponse{}
	_ = json.Unmarshal(body, &response)

	if response.Code == "invalid_token" {
		return errors.NewInvalidTokenError(response.Description)
	}

	return errors.NewHTTPError(statusCode, response.Code, response.Description)
}

func NewUAAGateway(config coreconfig.Reader, ui terminal.UI, logger trace.Printer, envDialTimeout string) Gateway {
	return Gateway{
		errHandler:      uaaErrorHandler,
		config:          config,
		PollingThrottle: DefaultPollingThrottle,
		warnings:        &[]string{},
		Clock:           time.Now,
		ui:              ui,
		logger:          logger,
		PollingEnabled:  false,
		DialTimeout:     dialTimeout(envDialTimeout),
	}
}
