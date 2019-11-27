// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
)

// RPC configurations.
type RPC struct {
	// MaxConnectionCount is the max gRPC connections that will be established with
	// each tikv-server.
	MaxConnectionCount uint

	// GrpcKeepAliveTime is the duration of time after which if the client doesn't see
	// any activity it pings the server to see if the transport is still alive.
	GrpcKeepAliveTime time.Duration

	// GrpcKeepAliveTimeout is the duration of time for which the client waits after having
	// pinged for keepalive check and if no activity is seen even after that the connection
	// is closed.
	GrpcKeepAliveTimeout time.Duration

	// GrpcMaxSendMsgSize set max gRPC request message size sent to server. If any request message size is larger than
	// current value, an error will be reported from gRPC.
	GrpcMaxSendMsgSize int

	// GrpcMaxCallMsgSize set max gRPC receive message size received from server. If any message size is larger than
	// current value, an error will be reported from gRPC.
	GrpcMaxCallMsgSize int

	// The value for initial window size on a gRPC stream.
	GrpcInitialWindowSize int

	// The value for initial windows size on a gRPC connection.
	GrpcInitialConnWindowSize int32

	// The max time to establish a gRPC connection.
	DialTimeout time.Duration

	// For requests that read/write several key-values.
	ReadTimeoutShort time.Duration

	// For requests that may need scan region.
	ReadTimeoutMedium time.Duration

	// For requests that may need scan region multiple times.
	ReadTimeoutLong time.Duration

	// The flag to enable open tracing.
	EnableOpenTracing bool

	// Batch system configurations.
	Batch Batch

	Security Security
}

// DefaultRPC returns the default RPC config.
func DefaultRPC() RPC {
	return RPC{
		MaxConnectionCount:        16,
		GrpcKeepAliveTime:         10 * time.Second,
		GrpcKeepAliveTimeout:      3 * time.Second,
		GrpcMaxSendMsgSize:        1<<31 - 1,
		GrpcMaxCallMsgSize:        1<<31 - 1,
		GrpcInitialWindowSize:     1 << 30,
		GrpcInitialConnWindowSize: 1 << 30,
		DialTimeout:               5 * time.Second,
		ReadTimeoutShort:          20 * time.Second,
		ReadTimeoutMedium:         60 * time.Second,
		ReadTimeoutLong:           150 * time.Second,
		EnableOpenTracing:         false,

		Batch:    DefaultBatch(),
		Security: DefaultSecurity(),
	}
}

// Batch contains configurations for message batch.
type Batch struct {
	// MaxBatchSize is the max batch size when calling batch commands API. Set 0 to
	// turn off message batch.
	MaxBatchSize uint

	// OverloadThreshold is a threshold of TiKV load. If TiKV load is greater than
	// this, TiDB will wait for a while to avoid little batch.
	OverloadThreshold uint

	// MaxWaitSize is the max wait size for batch.
	MaxWaitSize uint

	// MaxWaitTime  is the max wait time for batch.
	MaxWaitTime time.Duration
}

// DefaultBatch returns the default Batch config.
func DefaultBatch() Batch {
	return Batch{
		MaxBatchSize:      0,
		OverloadThreshold: 200,
		MaxWaitSize:       8,
		MaxWaitTime:       0,
	}
}

// Security is SSL configuration.
type Security struct {
	SSLCA   string `toml:"ssl-ca" json:"ssl-ca"`
	SSLCert string `toml:"ssl-cert" json:"ssl-cert"`
	SSLKey  string `toml:"ssl-key" json:"ssl-key"`
}

// ToTLSConfig generates tls's config based on security section of the config.
func (s *Security) ToTLSConfig() (*tls.Config, error) {
	var tlsConfig *tls.Config
	if len(s.SSLCA) != 0 {
		var certificates = make([]tls.Certificate, 0)
		if len(s.SSLCert) != 0 && len(s.SSLKey) != 0 {
			// Load the client certificates from disk
			certificate, err := tls.LoadX509KeyPair(s.SSLCert, s.SSLKey)
			if err != nil {
				return nil, errors.Errorf("could not load client key pair: %s", err)
			}
			certificates = append(certificates, certificate)
		}

		// Create a certificate pool from the certificate authority
		certPool := x509.NewCertPool()
		ca, err := ioutil.ReadFile(s.SSLCA)
		if err != nil {
			return nil, errors.Errorf("could not read ca certificate: %s", err)
		}

		// Append the certificates from the CA
		if !certPool.AppendCertsFromPEM(ca) {
			return nil, errors.New("failed to append ca certs")
		}

		tlsConfig = &tls.Config{
			Certificates: certificates,
			RootCAs:      certPool,
		}
	}

	return tlsConfig, nil
}

// DefaultSecurity returns the default Security config.
func DefaultSecurity() Security {
	return Security{}
}
