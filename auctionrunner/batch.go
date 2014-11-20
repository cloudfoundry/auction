package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
)

type Batch struct {
	startAuctions []auctiontypes.StartAuction
	stopAuctions  []auctiontypes.StopAuction
	lock          *sync.Mutex
	HasWork       chan struct{}
	timeProvider  timeprovider.TimeProvider
}

func NewBatch(timeProvider timeprovider.TimeProvider) *Batch {
	return &Batch{
		startAuctions: []auctiontypes.StartAuction{},
		stopAuctions:  []auctiontypes.StopAuction{},
		lock:          &sync.Mutex{},
		timeProvider:  timeProvider,
		HasWork:       make(chan struct{}, 1),
	}
}

func (b *Batch) AddLRPStartAuction(start models.LRPStartAuction) {
	b.lock.Lock()
	b.startAuctions = append(b.startAuctions, auctiontypes.StartAuction{
		LRPStartAuction: start,
		QueueTime:       b.timeProvider.Time(),
	})
	b.lock.Unlock()
	b.claimToHaveWork()
}

func (b *Batch) AddLRPStopAuction(stop models.LRPStopAuction) {
	b.lock.Lock()
	b.stopAuctions = append(b.stopAuctions, auctiontypes.StopAuction{
		LRPStopAuction: stop,
		QueueTime:      b.timeProvider.Time(),
	})
	b.lock.Unlock()
	b.claimToHaveWork()
}

func (b *Batch) ResubmitStartAuctions(starts []auctiontypes.StartAuction) {
	b.lock.Lock()
	b.startAuctions = append(starts, b.startAuctions...)
	b.lock.Unlock()
	b.claimToHaveWork()
}

func (b *Batch) ResubmitStopAuctions(stops []auctiontypes.StopAuction) {
	b.lock.Lock()
	b.stopAuctions = append(stops, b.stopAuctions...)
	b.lock.Unlock()
	b.claimToHaveWork()
}

func (b *Batch) DedupeAndDrain() ([]auctiontypes.StartAuction, []auctiontypes.StopAuction) {
	b.lock.Lock()
	startAuctions := b.startAuctions
	stopAuctions := b.stopAuctions
	b.startAuctions = []auctiontypes.StartAuction{}
	b.stopAuctions = []auctiontypes.StopAuction{}
	b.lock.Unlock()

	dedupedStartAuctions := []auctiontypes.StartAuction{}
	presentStartAuctions := map[string]bool{}
	for _, startAuction := range startAuctions {
		id := startAuction.Identifier()
		if presentStartAuctions[id] {
			continue
		}
		presentStartAuctions[id] = true
		dedupedStartAuctions = append(dedupedStartAuctions, startAuction)
	}

	dedupedStopAuctions := []auctiontypes.StopAuction{}
	presentStopAuctions := map[string]bool{}
	for _, stopAuction := range stopAuctions {
		id := stopAuction.Identifier()
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
