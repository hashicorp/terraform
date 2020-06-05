package blobs

type AccessTier string

var (
	Archive AccessTier = "Archive"
	Cool    AccessTier = "Cool"
	Hot     AccessTier = "Hot"
)

type ArchiveStatus string

var (
	None                   ArchiveStatus = ""
	RehydratePendingToCool ArchiveStatus = "rehydrate-pending-to-cool"
	RehydratePendingToHot  ArchiveStatus = "rehydrate-pending-to-hot"
)

type BlockListType string

var (
	All         BlockListType = "all"
	Committed   BlockListType = "committed"
	Uncommitted BlockListType = "uncommitted"
)

type Block struct {
	// The base64-encoded Block ID
	Name string `xml:"Name"`

	// The size of the Block in Bytes
	Size int64 `xml:"Size"`
}

type BlobType string

var (
	AppendBlob BlobType = "AppendBlob"
	BlockBlob  BlobType = "BlockBlob"
	PageBlob   BlobType = "PageBlob"
)

type CommittedBlocks struct {
	Blocks []Block `xml:"Block"`
}

type CopyStatus string

var (
	Aborted CopyStatus = "aborted"
	Failed  CopyStatus = "failed"
	Pending CopyStatus = "pending"
	Success CopyStatus = "success"
)

type LeaseDuration string

var (
	Fixed    LeaseDuration = "fixed"
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

type UncommittedBlocks struct {
	Blocks []Block `xml:"Block"`
}
