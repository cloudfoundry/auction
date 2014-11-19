package auctionrunner

import (
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type Batch struct {
	startAuctions []models.LRPStartAuction
	stopAuctions  []models.LRPStopAuction
	lock          *sync.Mutex
	HasWork       chan struct{}
}

func NewBatch() *Batch {
	return &Batch{
		startAuctions: []models.LRPStartAuction{},
		stopAuctions:  []models.LRPStopAuction{},
		lock:          &sync.Mutex{},
		HasWork:       make(chan struct{}, 1),
	}
}

func (b *Batch) AddLRPStartAuction(start models.LRPStartAuction) {
	b.lock.Lock()
	b.startAuctions = append(b.startAuctions, start)
	b.lock.Unlock()
	b.claimToHaveWork()
}

func (b *Batch) AddLRPStopAuction(start models.LRPStopAuction) {
	b.lock.Lock()
	b.stopAuctions = append(b.stopAuctions, start)
	b.lock.Unlock()
	b.claimToHaveWork()
}

func (b *Batch) DedupeAndDrain() ([]models.LRPStartAuction, []models.LRPStopAuction) {
	b.lock.Lock()
	startAuctions := b.startAuctions
	stopAuctions := b.stopAuctions
	b.startAuctions = []models.LRPStartAuction{}
	b.stopAuctions = []models.LRPStopAuction{}
	b.lock.Unlock()

	dedupedStartAuctions := []models.LRPStartAuction{}
	presentStartAuctions := map[string]bool{}
	for _, startAuction := range startAuctions {
		id := fmt.Sprintf("%s.%d.%s", startAuction.DesiredLRP.ProcessGuid, startAuction.Index, startAuction.InstanceGuid)
		if presentStartAuctions[id] {
			continue
		}
		presentStartAuctions[id] = true
		dedupedStartAuctions = append(dedupedStartAuctions, startAuction)
	}

	dedupedStopAuctions := []models.LRPStopAuction{}
	presentStopAuctions := map[string]bool{}
	for _, stopAuction := range stopAuctions {
		id := fmt.Sprintf("%s.%d", stopAuction.ProcessGuid, stopAuction.Index)
		if presentStopAuctions[id] {
			continue
		}
		presentStopAuctions[id] = true
		dedupedStopAuctions = append(dedupedStopAuctions, stopAuction)
	}

	return dedupedStartAuctions, dedupedStopAuctions
}

func (b *Batch) claimToHaveWork() {
	select {
	case b.HasWork <- struct{}{}:
	default:
	}
}
