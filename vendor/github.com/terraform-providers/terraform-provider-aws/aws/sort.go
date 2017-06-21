package aws

import (
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type imageSort []*ec2.Image
type snapshotSort []*ec2.Snapshot

func (a imageSort) Len() int {
	return len(a)
}

func (a imageSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a imageSort) Less(i, j int) bool {
	itime, _ := time.Parse(time.RFC3339, *a[i].CreationDate)
	jtime, _ := time.Parse(time.RFC3339, *a[j].CreationDate)
	return itime.Unix() < jtime.Unix()
}

// Sort images by creation date, in descending order.
func sortImages(images []*ec2.Image) []*ec2.Image {
	sortedImages := images
	sort.Sort(sort.Reverse(imageSort(sortedImages)))
	return sortedImages
}

func (a snapshotSort) Len() int {
	return len(a)
}

func (a snapshotSort) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a snapshotSort) Less(i, j int) bool {
	itime := *a[i].StartTime
	jtime := *a[j].StartTime
	return itime.Unix() < jtime.Unix()
}

// Sort snapshots by creation date, in descending order.
func sortSnapshots(snapshots []*ec2.Snapshot) []*ec2.Snapshot {
	sortedSnapshots := snapshots
	sort.Sort(sort.Reverse(snapshotSort(sortedSnapshots)))
	return sortedSnapshots
}
