package openstack

import (
	"sort"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
)

// blockStorageV3SnapshotSort represents a sortable slice of block storage
// v3 snapshots.
type blockStorageV3SnapshotSort []snapshots.Snapshot

func (snaphot blockStorageV3SnapshotSort) Len() int {
	return len(snaphot)
}

func (snaphot blockStorageV3SnapshotSort) Swap(i, j int) {
	snaphot[i], snaphot[j] = snaphot[j], snaphot[i]
}

func (snaphot blockStorageV3SnapshotSort) Less(i, j int) bool {
	itime := snaphot[i].CreatedAt
	jtime := snaphot[j].CreatedAt
	return itime.Unix() < jtime.Unix()
}

func dataSourceBlockStorageV3MostRecentSnapshot(snapshots []snapshots.Snapshot) snapshots.Snapshot {
	sortedSnapshots := snapshots
	sort.Sort(blockStorageV3SnapshotSort(sortedSnapshots))
	return sortedSnapshots[len(sortedSnapshots)-1]
}
