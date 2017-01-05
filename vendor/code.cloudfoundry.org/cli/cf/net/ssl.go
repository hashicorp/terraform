package net

import (
	"crypto/tls"
	"crypto/x509"
)

func NewTLSConfig(trustedCerts []tls.Certificate, disableSSL bool) (TLSConfig *tls.Config) {
	TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS10,
	}

	if len(trustedCerts) > 0 {
		certPool := x509.NewCertPool()
		for _, tlsCert := range trustedCerts {
			cert, _ := x509.ParseCertificate(tlsCert.Certificate[0])
			certPool.AddCert(cert)
		}
		TLSConfig.RootCAs = certPool
	}

	TLSConfig.InsecureSkipVerify = disableSSL

	return
}
