package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/timeprovider"
)

type Batch struct {
	lrpStartAuctions []auctiontypes.LRPStartAuction
	taskAuctions     []auctiontypes.TaskAuction
	lock             *sync.Mutex
	HasWork          chan struct{}
	timeProvider     timeprovider.TimeProvider
}

func NewBatch(timeProvider timeprovider.TimeProvider) *Batch {
	return &Batch{
		lrpStartAuctions: []auctiontypes.LRPStartAuction{},
		lock:             &sync.Mutex{},
		timeProvider:     timeProvider,
		HasWork:          make(chan struct{}, 1),
	}
}

func (b *Batch) AddLRPStart(start models.LRPStart) {
	b.lock.Lock()
	b.lrpStartAuctions = append(b.lrpStartAuctions, auctiontypes.LRPStartAuction{
		LRPStart: start,
		AuctionRecord: auctiontypes.AuctionRecord{
			QueueTime: b.timeProvider.Now(),
		},
	})
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) AddTask(task models.Task) {
	b.lock.Lock()
	b.taskAuctions = append(b.taskAuctions, auctiontypes.TaskAuction{
		Task: task,
		AuctionRecord: auctiontypes.AuctionRecord{
			QueueTime: b.timeProvider.Now(),
		},
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

func (b *Batch) ResubmitTaskAuctions(tasks []auctiontypes.TaskAuction) {
	b.lock.Lock()
	b.taskAuctions = append(tasks, b.taskAuctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) DedupeAndDrain() ([]auctiontypes.LRPStartAuction, []auctiontypes.TaskAuction) {
	b.lock.Lock()
	lrpStartAuctions := b.lrpStartAuctions
	taskAuctions := b.taskAuctions
	b.lrpStartAuctions = []auctiontypes.LRPStartAuction{}
	b.taskAuctions = []auctiontypes.TaskAuction{}
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

	return dedupedLRPStartAuctions, dedupedTaskAuctions
}

func (b *Batch) claimToHaveWork() {
	select {
	case b.HasWork <- struct{}{}:
	default:
	}
}
