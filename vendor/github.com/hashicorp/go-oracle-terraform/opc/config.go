package opc

import (
	"net/http"
	"net/url"
)

type Config struct {
	Username       *string
	Password       *string
	IdentityDomain *string
	APIEndpoint    *url.URL
	MaxRetries     *int
	LogLevel       LogLevelType
	Logger         Logger
	HTTPClient     *http.Client
}

func NewConfig() *Config {
	return &Config{}
}
