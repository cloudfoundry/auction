package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

type SortableLRPAuctions []auctiontypes.LRPAuction

func (a SortableLRPAuctions) Len() int {
	return len(a)
}

func (a SortableLRPAuctions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortableLRPAuctions) Less(i, j int) bool {
	if a[i].Index == a[j].Index {
		return a[i].DesiredLRP.MemoryMB > a[j].DesiredLRP.MemoryMB
	}

	return a[i].Index < a[j].Index
}
