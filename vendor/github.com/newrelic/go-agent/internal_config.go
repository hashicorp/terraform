package newrelic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/newrelic/go-agent/internal"
	"github.com/newrelic/go-agent/internal/logger"
	"github.com/newrelic/go-agent/internal/utilization"
)

func copyDestConfig(c AttributeDestinationConfig) AttributeDestinationConfig {
	cp := c
	if nil != c.Include {
		cp.Include = make([]string, len(c.Include))
		copy(cp.Include, c.Include)
	}
	if nil != c.Exclude {
		cp.Exclude = make([]string, len(c.Exclude))
		copy(cp.Exclude, c.Exclude)
	}
	return cp
}

func copyConfigReferenceFields(cfg Config) Config {
	cp := cfg
	if nil != cfg.Labels {
		cp.Labels = make(map[string]string, len(cfg.Labels))
		for key, val := range cfg.Labels {
			cp.Labels[key] = val
		}
	}
	if nil != cfg.ErrorCollector.IgnoreStatusCodes {
		ignored := make([]int, len(cfg.ErrorCollector.IgnoreStatusCodes))
		copy(ignored, cfg.ErrorCollector.IgnoreStatusCodes)
		cp.ErrorCollector.IgnoreStatusCodes = ignored
	}

	cp.Attributes = copyDestConfig(cfg.Attributes)
	cp.ErrorCollector.Attributes = copyDestConfig(cfg.ErrorCollector.Attributes)
	cp.TransactionEvents.Attributes = copyDestConfig(cfg.TransactionEvents.Attributes)
	cp.TransactionTracer.Attributes = copyDestConfig(cfg.TransactionTracer.Attributes)

	return cp
}

const (
	agentLanguage = "go"
)

func transportSetting(t http.RoundTripper) interface{} {
	if nil == t {
		return nil
	}
	return fmt.Sprintf("%T", t)
}

func loggerSetting(lg Logger) interface{} {
	if nil == lg {
		return nil
	}
	if _, ok := lg.(logger.ShimLogger); ok {
		return nil
	}
	return fmt.Sprintf("%T", lg)
}

const (
	// https://source.datanerd.us/agents/agent-specs/blob/master/Custom-Host-Names.md
	hostByteLimit = 255
)

type settings Config

func (s settings) MarshalJSON() ([]byte, error) {
	c := Config(s)
	transport := c.Transport
	c.Transport = nil
	logger := c.Logger
	c.Logger = nil

	js, err := json.Marshal(c)
	if nil != err {
		return nil, err
	}
	fields := make(map[string]interface{})
	err = json.Unmarshal(js, &fields)
	if nil != err {
		return nil, err
	}
	// The License field is not simply ignored by adding the `json:"-"` tag
	// to it since we want to allow consumers to populate Config from JSON.
	delete(fields, `License`)
	fields[`Transport`] = transportSetting(transport)
	fields[`Logger`] = loggerSetting(logger)
	return json.Marshal(fields)
}

func configConnectJSONInternal(c Config, pid int, util *utilization.Data, e internal.Environment, version string) ([]byte, error) {
	return json.Marshal([]interface{}{struct {
		Pid             int                  `json:"pid"`
		Language        string               `json:"language"`
		Version         string               `json:"agent_version"`
		Host            string               `json:"host"`
		HostDisplayName string               `json:"display_host,omitempty"`
		Settings        interface{}          `json:"settings"`
		AppName         []string             `json:"app_name"`
		HighSecurity    bool                 `json:"high_security"`
		Labels          internal.Labels      `json:"labels,omitempty"`
		Environment     internal.Environment `json:"environment"`
		Identifier      string               `json:"identifier"`
		Util            *utilization.Data    `json:"utilization"`
	}{
		Pid:             pid,
		Language:        agentLanguage,
		Version:         version,
		Host:            internal.StringLengthByteLimit(util.Hostname, hostByteLimit),
		HostDisplayName: internal.StringLengthByteLimit(c.HostDisplayName, hostByteLimit),
		Settings:        (settings)(c),
		AppName:         strings.Split(c.AppName, ";"),
		HighSecurity:    c.HighSecurity,
		Labels:          internal.Labels(c.Labels),
		Environment:     e,
		// This identifier field is provided to avoid:
		// https://newrelic.atlassian.net/browse/DSCORE-778
		//
		// This identifier is used by the collector to look up the real
		// agent. If an identifier isn't provided, the collector will
		// create its own based on the first appname, which prevents a
		// single daemon from connecting "a;b" and "a;c" at the same
		// time.
		//
		// Providing the identifier below works around this issue and
		// allows users more flexibility in using application rollups.
		Identifier: c.AppName,
		Util:       util,
	}})
}

func configConnectJSON(c Config) ([]byte, error) {
	env := internal.NewEnvironment()
	util := utilization.Gather(utilization.Config{
		DetectAWS:         c.Utilization.DetectAWS,
		DetectDocker:      c.Utilization.DetectDocker,
		LogicalProcessors: c.Utilization.LogicalProcessors,
		TotalRAMMIB:       c.Utilization.TotalRAMMIB,
		BillingHostname:   c.Utilization.BillingHostname,
	}, c.Logger)
	return configConnectJSONInternal(c, os.Getpid(), util, env, Version)
}
