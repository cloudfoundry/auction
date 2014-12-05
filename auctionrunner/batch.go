package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
)

type Batch struct {
	lrpStartAuctions []auctiontypes.LRPStartAuction
	lrpStopAuctions  []auctiontypes.LRPStopAuction
	lock             *sync.Mutex
	HasWork          chan struct{}
	timeProvider     timeprovider.TimeProvider
}

func NewBatch(timeProvider timeprovider.TimeProvider) *Batch {
	return &Batch{
		lrpStartAuctions: []auctiontypes.LRPStartAuction{},
		lrpStopAuctions:  []auctiontypes.LRPStopAuction{},
		lock:             &sync.Mutex{},
		timeProvider:     timeProvider,
		HasWork:          make(chan struct{}, 1),
	}
}

func (b *Batch) AddLRPStartAuction(start models.LRPStartAuction) {
	b.lock.Lock()
	b.lrpStartAuctions = append(b.lrpStartAuctions, auctiontypes.LRPStartAuction{
		LRPStartAuction: start,
		QueueTime:       b.timeProvider.Now(),
	})
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) AddLRPStopAuction(stop models.LRPStopAuction) {
	b.lock.Lock()
	b.lrpStopAuctions = append(b.lrpStopAuctions, auctiontypes.LRPStopAuction{
		LRPStopAuction: stop,
		QueueTime:      b.timeProvider.Now(),
	})
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) ResubmitStartAuctions(starts []auctiontypes.LRPStartAuction) {
	b.lock.Lock()
	b.lrpStartAuctions = append(starts, b.lrpStartAuctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) ResubmitStopAuctions(stops []auctiontypes.LRPStopAuction) {
	b.lock.Lock()
	b.lrpStopAuctions = append(stops, b.lrpStopAuctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) DedupeAndDrain() ([]auctiontypes.LRPStartAuction, []auctiontypes.LRPStopAuction) {
	b.lock.Lock()
	lrpStartAuctions := b.lrpStartAuctions
	lrpStopAuctions := b.lrpStopAuctions
	b.lrpStartAuctions = []auctiontypes.LRPStartAuction{}
	b.lrpStopAuctions = []auctiontypes.LRPStopAuction{}
	select {
	case <-b.HasWork:
	default:
	}
	b.lock.Unlock()

	dedupedLRPStartAuctions := []auctiontypes.LRPStartAuction{}
	presentLRPStartAuctions := map[string]bool{}
	for _, startAuction := range lrpStartAuctions {
		id := startAuction.Identifier()
		if presentLRPStartAuctions[id] {
			continue
		}
		presentLRPStartAuctions[id] = true
		dedupedLRPStartAuctions = append(dedupedLRPStartAuctions, startAuction)
	}

	dedupedLRPStopAuctions := []auctiontypes.LRPStopAuction{}
	presentLRPStopAuctions := map[string]bool{}
	for _, stopAuction := range lrpStopAuctions {
		id := stopAuction.Identifier()
		if presentLRPStopAuctions[id] {
			continue
		}
		presentLRPStopAuctions[id] = true
		dedupedLRPStopAuctions = append(dedupedLRPStopAuctions, stopAuction)
	}

	return dedupedLRPStartAuctions, dedupedLRPStopAuctions
}

func (b *Batch) claimToHaveWork() {
	select {
	case b.HasWork <- struct{}{}:
	default:
	}
}
