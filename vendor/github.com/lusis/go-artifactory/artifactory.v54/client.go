package artifactory

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"
)

// ClientConfig is the configuration for an ArtifactoryClient
type ClientConfig struct {
	BaseURL    string
	Username   string
	Password   string
	Token      string
	AuthMethod string
	VerifySSL  bool
	Client     *http.Client
	Transport  *http.Transport
}

// Client is a client for interacting with Artifactory
type Client struct {
	Client    *http.Client
	Config    *ClientConfig
	Transport *http.Transport
}

// NewClientFromEnv returns a new ArtifactoryClient the is automatically configured from environment variables
func NewClientFromEnv() (*Client, error) {
	config, err := clientConfigFrom("environment")
	if err != nil {
		return nil, err
	}

	client := NewClient(config)

	return &client, nil
}

// NewClient returns a new ArtifactoryClient with the provided ClientConfig
func NewClient(config *ClientConfig) Client {
	verifySSL := func() bool {
		return !config.VerifySSL
	}
	if config.Transport == nil {
		config.Transport = &http.Transport{}
	}
	config.Transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: verifySSL()}
	if config.Client == nil {
		config.Client = &http.Client{}
	}
	config.Client.Transport = config.Transport
	return Client{Client: config.Client, Config: config, Transport: config.Transport}
}

func clientConfigFrom(from string) (*ClientConfig, error) {
	conf := ClientConfig{}
	switch from {
	case "environment":
		if os.Getenv("ARTIFACTORY_URL") == "" {
			return nil, errors.New("You must set the environment variable ARTIFACTORY_URL")
		}

		conf.BaseURL = os.Getenv("ARTIFACTORY_URL")
		if os.Getenv("ARTIFACTORY_TOKEN") == "" {
			if os.Getenv("ARTIFACTORY_USERNAME") == "" || os.Getenv("ARTIFACTORY_PASSWORD") == "" {
				return nil, errors.New("You must set the environment variables ARTIFACTORY_USERNAME/ARTIFACTORY_PASSWORD")
			}

			conf.AuthMethod = "basic"
		} else {
			conf.AuthMethod = "token"
		}
	}
	if conf.AuthMethod == "token" {
		conf.Token = os.Getenv("ARTIFACTORY_TOKEN")
	} else {
		conf.Username = os.Getenv("ARTIFACTORY_USERNAME")
		conf.Password = os.Getenv("ARTIFACTORY_PASSWORD")
	}
	return &conf, nil
}
