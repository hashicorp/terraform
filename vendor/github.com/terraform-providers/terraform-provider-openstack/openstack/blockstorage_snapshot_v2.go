package openstack

import (
	"sort"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v2/snapshots"
)

// blockStorageV2SnapshotSort represents a sortable slice of block storage
// v2 snapshots.
type blockStorageV2SnapshotSort []snapshots.Snapshot

func (snaphot blockStorageV2SnapshotSort) Len() int {
	return len(snaphot)
}

func (snaphot blockStorageV2SnapshotSort) Swap(i, j int) {
	snaphot[i], snaphot[j] = snaphot[j], snaphot[i]
}

func (snaphot blockStorageV2SnapshotSort) Less(i, j int) bool {
	itime := snaphot[i].CreatedAt
	jtime := snaphot[j].CreatedAt
	return itime.Unix() < jtime.Unix()
}

func dataSourceBlockStorageV2MostRecentSnapshot(snapshots []snapshots.Snapshot) snapshots.Snapshot {
	sortedSnapshots := snapshots
	sort.Sort(blockStorageV2SnapshotSort(sortedSnapshots))
	return sortedSnapshots[len(sortedSnapshots)-1]
}
