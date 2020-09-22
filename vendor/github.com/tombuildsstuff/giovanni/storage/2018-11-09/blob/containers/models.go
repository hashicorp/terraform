package containers

import "github.com/Azure/go-autorest/autorest"

type AccessLevel string

var (
	// Blob specifies public read access for blobs.
	// Blob data within this container can be read via anonymous request,
	// but container data is not available.
	// Clients cannot enumerate blobs within the container via anonymous request.
	Blob AccessLevel = "blob"

	// Container specifies full public read access for container and blob data.
	// Clients can enumerate blobs within the container via anonymous request,
	// but cannot enumerate containers within the storage account.
	Container AccessLevel = "container"

	// Private specifies that container data is private to the account owner
	Private AccessLevel = ""
)

type ContainerProperties struct {
	autorest.Response

	AccessLevel           AccessLevel
	LeaseStatus           LeaseStatus
	LeaseState            LeaseState
	LeaseDuration         *LeaseDuration
	MetaData              map[string]string
	HasImmutabilityPolicy bool
	HasLegalHold          bool
}

type Dataset string

var (
	Copy             Dataset = "copy"
	Deleted          Dataset = "deleted"
	MetaData         Dataset = "metadata"
	Snapshots        Dataset = "snapshots"
	UncommittedBlobs Dataset = "uncommittedblobs"
)

type ErrorResponse struct {
	Code    *string `xml:"Code"`
	Message *string `xml:"Message"`
}

type LeaseDuration string

var (
	// If this lease is for a Fixed Duration
	Fixed LeaseDuration = "fixed"

	// If this lease is for an Indefinite Duration
	Infinite LeaseDuration = "infinite"
)

type LeaseState string

var (
	Available LeaseState = "available"
	Breaking  LeaseState = "breaking"
	Broken    LeaseState = "broken"
	Expired   LeaseState = "expired"
	Leased    LeaseState = "leased"
)

type LeaseStatus string

var (
	Locked   LeaseStatus = "locked"
	Unlocked LeaseStatus = "unlocked"
)
