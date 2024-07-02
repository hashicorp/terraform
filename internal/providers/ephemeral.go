// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"time"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// OpenEphemeralRequest represents the arguments for the OpenEphemeral
// operation on a provider.
type OpenEphemeralRequest struct {
	// TypeName is the type of ephemeral resource to open. This should
	// only be one of the type names previously reported in the provider's
	// schema.
	TypeName string

	// Config is an object-typed value representing the configuration for
	// the ephemeral resource instance that the caller is trying to open.
	//
	// The object type of this value always conforms to the resource type
	// schema's implied type, and uses null values to represent attributes
	// that were not explicitly assigned in the configuration block.
	// Computed-only attributes are always null in the configuration, because
	// they can be set only in the reponse.
	Config cty.Value
}

// OpenEphemeralRequest represents the response from an OpenEphemeral
// operation on a provider.
type OpenEphemeralResponse struct {
	// Deferred, if present, signals that the provider doesn't have enough
	// information to open this ephemeral resource instance.
	//
	// This implies that any other side-effect-performing object must have
	// its planning deferred if its planning operation indirectly depends on
	// this ephemeral resource result. For example, if a provider configuration
	// refers to an ephemeral resource whose opening is deferred then the
	// affected provider configuration must not be instantiated and any
	// resource instances that belong to it must have their planning immediately
	// deferred.
	Deferred *Deferred

	// Result is an object-typed value representing the newly-opened session
	// with the opened ephemeral object.
	//
	// The object type of this value always conforms to the resource type
	// schema's implied type. Unknown values are forbidden unless the
	// Deferred field is set, in which case the Result represents the provider's
	// best approximation of the final object using unknown values in any
	// location where a final value cannot be predicted.
	Result cty.Value

	// InternalContext is any internal data needed by the provider to
	// perform a subsequent [Interface.CloseEphemeral] request for the same
	// object. The provider may choose any encoding format to represent the
	// needed data, because Terraform Core treats this field as opaque.
	//
	// Providers should aim to keep this data relatively compact to minimize
	// overhead. Although Terraform Core does not enforce a specific limit
	// just for this field, it would be very unusual for the internal context
	// to be more than 256 bytes in size, and in most cases it should be
	// on the order of only tens of bytes. For example, a lease ID for the
	// remote system is a reasonable thing to encode here.
	//
	// Because ephemeral resource instances never outlive a single Terraform
	// Core phase, it's guaranteed that a CloseEphemeral request will be
	// received by exactly the same plugin instance that returned this
	// value, and so it's valid for this to refer to in-memory state belonging
	// to the provider instance.
	InternalContext []byte

	// Renew, if set, signals that the opened object has an inherent expration
	// time and so must be "renewed" if Terraform needs to use it beyond that
	// expiration time.
	//
	// If a provider sets this field then it may receive a subsequent
	// [Interface.RenewEphemeral] call, if Terraform expects to need the
	// object beyond the expiration time.
	Renew *EphemeralRenew

	// Diagnostics describes any problems encountered while opening the
	// ephemeral resource. If this contains errors then the other response
	// fields must be assumed invalid.
	Diagnostics tfdiags.Diagnostics
}

// EphemeralRenew describes when and how Terraform Core must request renewal
// of an ephemeral resource instance in order to continue using it.
type EphemeralRenew struct {
	// ExpireTime is the deadline before which Terraform must renew the
	// ephemeral resource instance. Terraform will make the renew request
	// at least one minute before the expiration time.
	ExpireTime time.Time

	// InternalContext is any internal data needed by the provider to
	// perform a subsequent [Interface.RenewEphemeral] request. The provider
	// may choose any encoding format to represent the needed data, because
	// Terraform Core treats this field as opaque.
	//
	// Providers should aim to keep this data relatively compact to minimize
	// overhead. Although Terraform Core does not enforce a specific limit
	// just for this field, it would be very unusual for the internal context
	// to be more than 256 bytes in size, and in most cases it should be
	// on the order of only tens of bytes. For example, a lease ID for the
	// remote system is a reasonable thing to encode here.
	//
	// Because ephemeral resource instances never outlive a single Terraform
	// Core phase, it's guaranteed that a RenewEphemeral request will be
	// received by exactly the same plugin instance that previously handled
	// the OpenEphemeral or RenewEphemeral request that produced this internal
	// context, and so it's valid for this to refer to in-memory state in the
	// provider object.
	InternalContext []byte
}

// RenewEphemeralRequest represents the arguments for the RenewEphemeral
// operation on a provider.
type RenewEphemeralRequest struct {
	// TypeName is the type of ephemeral resource being renewed. This should
	// only be one of the type names previously sent in a successful
	// [OpenEphemeralRequest].
	TypeName string

	// InternalContext echoes verbatim the value from the field of the same
	// name from the most recent [EphemeralRenew] object, received from either
	// an [OpenEphemeralResponse] or a [RenewEphemeralResponse] object.
	InternalContext []byte
}

// RenewEphemeralRequest represents the response from a RenewEphemeral
// operation on a provider.
type RenewEphemeralResponse struct {
	// RenewAgain, if set, describes a new expiration deadline for the
	// object, possibly causing a further call to [Interface.RenewEphemeral]
	// if Terraform needs to exceed the updated deadline.
	//
	// If this is not set then Terraform Core will not make any further
	// renewal requests for the remaining life of the object.
	RenewAgain *EphemeralRenew

	// Diagnostics describes any problems encountered while renewing the
	// ephemeral resource instance. If this contains errors then the other
	// response fields must be assumed invalid.
	//
	// Because renewals happen asynchronously from other uses of the
	// ephemeral object, it's unspecified whether a renewal error will block
	// any specific usage of the object. For example, a request using the
	// object might already be in progress when a renewal error occurs,
	// in which case that other request might also fail trying to use a
	// now-invalid object, or it might by chance succeed in completing its
	// operation before the ephemeral object truly expires.
	Diagnostics tfdiags.Diagnostics
}

// CloseEphemeralRequest represents the arguments for the CloseEphemeral
// operation on a provider.
type CloseEphemeralRequest struct {
	// TypeName is the type of ephemeral resource being closed. This should
	// only be one of the type names previously sent in a successful
	// [OpenEphemeralRequest].
	TypeName string

	// InternalContext echoes verbatim the value from the field of the same
	// name from the corresponding [OpenEphemeralResponse] object.
	InternalContext []byte
}

// CloseEphemeralRequest represents the response from a CloseEphemeral
// operation on a provider.
type CloseEphemeralResponse struct {
	// Diagnostics describes any problems encountered while closing the
	// ephemeral resource instance. If this contains errors then the other
	// response fields must be assumed invalid.
	//
	// If closing an ephemeral resource instance fails then it's unspecified
	// whether a corresponding remote object remains valid or not.
	//
	// Providers should make a best effort to treat the closure of an
	// already-expired ephemeral object as a success in order to exhibit
	// idemponent behavior for closing, but some remote systems do not allow
	// distinguishing that case from other error conditions.
	Diagnostics tfdiags.Diagnostics
}
