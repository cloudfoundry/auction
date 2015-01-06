package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

type SortableAuctions []auctiontypes.LRPAuction

func (a SortableAuctions) Len() int {
	return len(a)
}

func (a SortableAuctions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortableAuctions) Less(i, j int) bool {
	if a[i].Index == a[j].Index {
		return a[i].DesiredLRP.MemoryMB > a[j].DesiredLRP.MemoryMB
	}

	return a[i].Index < a[j].Index
}
