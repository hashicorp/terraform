package pq

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
)

// ssl generates a function to upgrade a net.Conn based on the "sslmode" and
// related settings. The function is nil when no upgrade should take place.
func ssl(o values) func(net.Conn) net.Conn {
	verifyCaOnly := false
	tlsConf := tls.Config{}
	switch mode := o.Get("sslmode"); mode {
	// "require" is the default.
	case "", "require":
		// We must skip TLS's own verification since it requires full
		// verification since Go 1.3.
		tlsConf.InsecureSkipVerify = true

		// From http://www.postgresql.org/docs/current/static/libpq-ssl.html:
		// Note: For backwards compatibility with earlier versions of PostgreSQL, if a
		// root CA file exists, the behavior of sslmode=require will be the same as
		// that of verify-ca, meaning the server certificate is validated against the
		// CA. Relying on this behavior is discouraged, and applications that need
		// certificate validation should always use verify-ca or verify-full.
		if _, err := os.Stat(o.Get("sslrootcert")); err == nil {
			verifyCaOnly = true
		} else {
			o.Set("sslrootcert", "")
		}
	case "verify-ca":
		// We must skip TLS's own verification since it requires full
		// verification since Go 1.3.
		tlsConf.InsecureSkipVerify = true
		verifyCaOnly = true
	case "verify-full":
		tlsConf.ServerName = o.Get("host")
	case "disable":
		return nil
	default:
		errorf(`unsupported sslmode %q; only "require" (default), "verify-full", "verify-ca", and "disable" supported`, mode)
	}

	sslClientCertificates(&tlsConf, o)
	sslCertificateAuthority(&tlsConf, o)
	sslRenegotiation(&tlsConf)

	return func(conn net.Conn) net.Conn {
		client := tls.Client(conn, &tlsConf)
		if verifyCaOnly {
			sslVerifyCertificateAuthority(client, &tlsConf)
		}
		return client
	}
}

// sslClientCertificates adds the certificate specified in the "sslcert" and
// "sslkey" settings, or if they aren't set, from the .postgresql directory
// in the user's home directory. The configured files must exist and have
// the correct permissions.
func sslClientCertificates(tlsConf *tls.Config, o values) {
	sslkey := o.Get("sslkey")
	sslcert := o.Get("sslcert")

	var cinfo, kinfo os.FileInfo
	var err error

	if sslcert != "" && sslkey != "" {
		// Check that both files exist. Note that we don't do any more extensive
		// checks than this (such as checking that the paths aren't directories);
		// LoadX509KeyPair() will take care of the rest.
		cinfo, err = os.Stat(sslcert)
		if err != nil {
			panic(err)
		}

		kinfo, err = os.Stat(sslkey)
		if err != nil {
			panic(err)
		}
	} else {
		// Automatically find certificates from ~/.postgresql
		sslcert, sslkey, cinfo, kinfo = sslHomeCertificates()

		if cinfo == nil || kinfo == nil {
			// No certificates to load
			return
		}
	}

	// The files must also have the correct permissions
	sslCertificatePermissions(cinfo, kinfo)

	cert, err := tls.LoadX509KeyPair(sslcert, sslkey)
	if err != nil {
		panic(err)
	}
	tlsConf.Certificates = []tls.Certificate{cert}
}

// sslCertificateAuthority adds the RootCA specified in the "sslrootcert" setting.
func sslCertificateAuthority(tlsConf *tls.Config, o values) {
	if sslrootcert := o.Get("sslrootcert"); sslrootcert != "" {
		tlsConf.RootCAs = x509.NewCertPool()

		cert, err := ioutil.ReadFile(sslrootcert)
		if err != nil {
			panic(err)
		}

		ok := tlsConf.RootCAs.AppendCertsFromPEM(cert)
		if !ok {
			errorf("couldn't parse pem in sslrootcert")
		}
	}
}

// sslHomeCertificates returns the path and stats of certificates in the current
// user's home directory.
func sslHomeCertificates() (cert, key string, cinfo, kinfo os.FileInfo) {
	user, err := user.Current()

	if err != nil {
		// user.Current() might fail when cross-compiling. We have to ignore the
		// error and continue without client certificates, since we wouldn't know
		// from where to load them.
		return
	}

	cert = filepath.Join(user.HomeDir, ".postgresql", "postgresql.crt")
	key = filepath.Join(user.HomeDir, ".postgresql", "postgresql.key")

	cinfo, err = os.Stat(cert)
	if err != nil {
		cinfo = nil
	}

	kinfo, err = os.Stat(key)
	if err != nil {
		kinfo = nil
	}

	return
}

// sslVerifyCertificateAuthority carries out a TLS handshake to the server and
// verifies the presented certificate against the CA, i.e. the one specified in
// sslrootcert or the system CA if sslrootcert was not specified.
func sslVerifyCertificateAuthority(client *tls.Conn, tlsConf *tls.Config) {
	err := client.Handshake()
	if err != nil {
		panic(err)
	}
	certs := client.ConnectionState().PeerCertificates
	opts := x509.VerifyOptions{
		DNSName:       client.ConnectionState().ServerName,
		Intermediates: x509.NewCertPool(),
		Roots:         tlsConf.RootCAs,
	}
	for i, cert := range certs {
		if i == 0 {
			continue
		}
		opts.Intermediates.AddCert(cert)
	}
	_, err = certs[0].Verify(opts)
	if err != nil {
		panic(err)
	}
}
