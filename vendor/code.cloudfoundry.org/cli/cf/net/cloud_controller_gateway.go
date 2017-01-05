package net

import (
	"encoding/json"
	"strconv"
	"time"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
)

type ccErrorResponse struct {
	Code        int
	Description string
}

const invalidTokenCode = 1000

func cloudControllerErrorHandler(statusCode int, body []byte) error {
	response := ccErrorResponse{}
	_ = json.Unmarshal(body, &response)

	if response.Code == invalidTokenCode {
		return errors.NewInvalidTokenError(response.Description)
	}

	return errors.NewHTTPError(statusCode, strconv.Itoa(response.Code), response.Description)
}

func NewCloudControllerGateway(config coreconfig.Reader, clock func() time.Time, ui terminal.UI, logger trace.Printer, envDialTimeout string) Gateway {
	return Gateway{
		errHandler:      cloudControllerErrorHandler,
		config:          config,
		PollingThrottle: DefaultPollingThrottle,
		warnings:        &[]string{},
		Clock:           clock,
		ui:              ui,
		logger:          logger,
		PollingEnabled:  true,
		DialTimeout:     dialTimeout(envDialTimeout),
	}
}
