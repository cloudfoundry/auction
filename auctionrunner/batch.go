package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
)

type Batch struct {
	lrpAuctions  []auctiontypes.LRPAuction
	taskAuctions []auctiontypes.TaskAuction
	lock         *sync.Mutex
	HasWork      chan struct{}
	timeProvider timeprovider.TimeProvider
}

func NewBatch(timeProvider timeprovider.TimeProvider) *Batch {
	return &Batch{
		lrpAuctions:  []auctiontypes.LRPAuction{},
		lock:         &sync.Mutex{},
		timeProvider: timeProvider,
		HasWork:      make(chan struct{}, 1),
	}
}

func (b *Batch) AddLRPStarts(starts []models.LRPStartRequest) {
	auctions := make([]auctiontypes.LRPAuction, 0, len(starts))
	now := b.timeProvider.Now()
	for _, start := range starts {
		for _, i := range start.Indices {
			auctions = append(auctions, auctiontypes.LRPAuction{
				DesiredLRP: start.DesiredLRP,
				Index:      int(i),
				AuctionRecord: auctiontypes.AuctionRecord{
					QueueTime: now,
				}})
		}
	}

	b.lock.Lock()
	b.lrpAuctions = append(b.lrpAuctions, auctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) AddTasks(tasks []models.Task) {
	auctions := make([]auctiontypes.TaskAuction, 0, len(tasks))
	now := b.timeProvider.Now()
	for _, t := range tasks {
		auctions = append(auctions, auctiontypes.TaskAuction{
			Task: t,
			AuctionRecord: auctiontypes.AuctionRecord{
				QueueTime: now,
			},
		})
	}

	b.lock.Lock()
	b.taskAuctions = append(b.taskAuctions, auctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) ResubmitStartAuctions(starts []auctiontypes.LRPAuction) {
	b.lock.Lock()
	b.lrpAuctions = append(starts, b.lrpAuctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) ResubmitTaskAuctions(tasks []auctiontypes.TaskAuction) {
	b.lock.Lock()
	b.taskAuctions = append(tasks, b.taskAuctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) DedupeAndDrain() ([]auctiontypes.LRPAuction, []auctiontypes.TaskAuction) {
	b.lock.Lock()
	lrpAuctions := b.lrpAuctions
	taskAuctions := b.taskAuctions
	b.lrpAuctions = []auctiontypes.LRPAuction{}
	b.taskAuctions = []auctiontypes.TaskAuction{}
	select {
	case <-b.HasWork:
	default:
	}
	b.lock.Unlock()

	dedupedLRPAuctions := []auctiontypes.LRPAuction{}
	presentLRPAuctions := map[string]bool{}
	for _, startAuction := range lrpAuctions {
		id := startAuction.Identifier()
		if presentLRPAuctions[id] {
			continue
		}
		presentLRPAuctions[id] = true
		dedupedLRPAuctions = append(dedupedLRPAuctions, startAuction)
	}

	dedupedTaskAuctions := []auctiontypes.TaskAuction{}
	presentTaskAuctions := map[string]bool{}
	for _, taskAuction := range taskAuctions {
		id := taskAuction.Identifier()
		if presentTaskAuctions[id] {
			continue
		}
		presentTaskAuctions[id] = true
		dedupedTaskAuctions = append(dedupedTaskAuctions, taskAuction)
	}

	return dedupedLRPAuctions, dedupedTaskAuctions
}

func (b *Batch) claimToHaveWork() {
	select {
	case b.HasWork <- struct{}{}:
	default:
	}
}
