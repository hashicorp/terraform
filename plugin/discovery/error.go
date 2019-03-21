package discovery

// Error is a type used to describe situations that the caller must handle
// since they indicate some form of user error.
//
// The functions and methods that return these specialized errors indicate so
// in their documentation. The Error type should not itself be used directly,
// but rather errors should be compared using the == operator with the
// error constants in this package.
//
// Values of this type are _not_ used when the error being reported is an
// operational error (server unavailable, etc) or indicative of a bug in
// this package or its caller.
type Error string

// ErrorNoSuitableVersion indicates that a suitable version (meeting given
// constraints) is not available.
const ErrorNoSuitableVersion = Error("no suitable version is available")

// ErrorNoVersionCompatible indicates that all of the available versions
// that otherwise met constraints are not compatible with the current
// version of Terraform.
const ErrorNoVersionCompatible = Error("no available version is compatible with this version of Terraform")

// ErrorVersionIncompatible indicates that all of the versions within the
// constraints are not compatible with the current version of Terrafrom, though
// there does exist a version outside of the constaints that is compatible.
const ErrorVersionIncompatible = Error("incompatible provider version")

// ErrorNoSuchProvider indicates that no provider exists with a name given
const ErrorNoSuchProvider = Error("no provider exists with the given name")

// ErrorNoVersionCompatibleWithPlatform indicates that all of the available
// versions that otherwise met constraints are not compatible with the
// requested platform
const ErrorNoVersionCompatibleWithPlatform = Error("no available version is compatible for the requested platform")

// ErrorMissingChecksumVerification indicates that either the provider
// distribution is missing the SHA256SUMS file or the checksum file does
// not contain a checksum for the binary plugin
const ErrorMissingChecksumVerification = Error("unable to verify checksum")

// ErrorChecksumVerification indicates that the current checksum of the
// provider plugin has changed since the initial release and is not trusted
// to download
const ErrorChecksumVerification = Error("unexpected plugin checksum")

// ErrorSignatureVerification indicates that the digital signature for a
// provider distribution could not be verified for one of the following
// reasons: missing signature file, missing public key, or the signature
// was not signed by any known key for the publisher
const ErrorSignatureVerification = Error("unable to verify signature")

func (err Error) Error() string {
	return string(err)
}
