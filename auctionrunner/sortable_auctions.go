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
		return a[i].DesiredLRP.MemoryMb > a[j].DesiredLRP.MemoryMb
	}

	return a[i].Index < a[j].Index
}

type SortableTaskAuctions []auctiontypes.TaskAuction

func (a SortableTaskAuctions) Len() int {
	return len(a)
}

func (a SortableTaskAuctions) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a SortableTaskAuctions) Less(i, j int) bool {
	return a[i].Task.MemoryMb > a[j].Task.MemoryMb
}
