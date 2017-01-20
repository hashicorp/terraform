// +build windows

package pq

import "os"

// sslCertificatePermissions checks the permissions on user-supplied certificate
// files. In libpq, this is a no-op on Windows.
func sslCertificatePermissions(cert, key os.FileInfo) {}
