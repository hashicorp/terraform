package shared

import (
	"fmt"
	"net"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

// ConnectionBlockSupersetSchema is a schema representing the superset of all
// possible arguments for "connection" blocks across all supported connection
// types.
//
// This currently lives here because we've not yet updated our communicator
// subsystem to be aware of schema itself. Once that is done, we can remove
// this and use a type-specific schema from the communicator to validate
// exactly what is expected for a given connection type.
var ConnectionBlockSupersetSchema = &configschema.Block{
	Attributes: map[string]*configschema.Attribute{
		// Common attributes for both connection types
		"host": {
			Type:     cty.String,
			Required: true,
		},
		"type": {
			Type:     cty.String,
			Optional: true,
		},
		"user": {
			Type:     cty.String,
			Optional: true,
		},
		"password": {
			Type:     cty.String,
			Optional: true,
		},
		"port": {
			Type:     cty.Number,
			Optional: true,
		},
		"timeout": {
			Type:     cty.String,
			Optional: true,
		},
		"script_path": {
			Type:     cty.String,
			Optional: true,
		},
		// For type=ssh only (enforced in ssh communicator)
		"target_platform": {
			Type:     cty.String,
			Optional: true,
		},
		"private_key": {
			Type:     cty.String,
			Optional: true,
		},
		"certificate": {
			Type:     cty.String,
			Optional: true,
		},
		"host_key": {
			Type:     cty.String,
			Optional: true,
		},
		"agent": {
			Type:     cty.Bool,
			Optional: true,
		},
		"agent_identity": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_host": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_host_key": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_port": {
			Type:     cty.Number,
			Optional: true,
		},
		"bastion_user": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_password": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_private_key": {
			Type:     cty.String,
			Optional: true,
		},
		"bastion_certificate": {
			Type:     cty.String,
			Optional: true,
		},

		// For type=winrm only (enforced in winrm communicator)
		"https": {
			Type:     cty.Bool,
			Optional: true,
		},
		"insecure": {
			Type:     cty.Bool,
			Optional: true,
		},
		"cacert": {
			Type:     cty.String,
			Optional: true,
		},
		"use_ntlm": {
			Type:     cty.Bool,
			Optional: true,
		},
	},
}

// IpFormat formats the IP correctly, so we don't provide IPv6 address in an IPv4 format during node communication. We return the ip parameter as is if it's an IPv4 address or a hostname.
func IpFormat(ip string) string {
	ipObj := net.ParseIP(ip)
	// Return the ip/host as is if it's either a hostname or an IPv4 address.
	if ipObj == nil || ipObj.To4() != nil {
		return ip
	}

	return fmt.Sprintf("[%s]", ip)
}
