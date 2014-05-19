package auctionrep

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type AuctionRepDelegate interface {
	RemainingResources() auctiontypes.Resources
	TotalResources() auctiontypes.Resources
	NumInstancesForAppGuid(guid string) int

	Reserve(instance auctiontypes.LRPAuctionInfo) error
	ReleaseReservation(instance auctiontypes.LRPAuctionInfo) error
	Claim(instance models.LRPStartAuction) error
}

//Used in simulation
type SimulationAuctionRepDelegate interface {
	AuctionRepDelegate
	SetLRPAuctionInfos(instances []auctiontypes.LRPAuctionInfo)
	LRPAuctionInfos() []auctiontypes.LRPAuctionInfo
}

type AuctionRep struct {
	guid     string
	delegate AuctionRepDelegate
	lock     *sync.Mutex
}

func New(guid string, delegate AuctionRepDelegate) *AuctionRep {
	return &AuctionRep{
		guid:     guid,
		delegate: delegate,
		lock:     &sync.Mutex{},
	}
}

func (rep *AuctionRep) Guid() string {
	return rep.guid
}

func (rep *AuctionRep) Score(instance auctiontypes.LRPAuctionInfo) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.delegate.RemainingResources()
	if !rep.hasRoomFor(instance, remaining) {
		return 0, auctiontypes.InsufficientResources
	}

	total := rep.delegate.TotalResources()
	nInstances := rep.delegate.NumInstancesForAppGuid(instance.AppGuid)

	return rep.score(remaining, total, nInstances), nil
}

func (rep *AuctionRep) ScoreThenTentativelyReserve(instance auctiontypes.LRPAuctionInfo) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.delegate.RemainingResources()
	if !rep.hasRoomFor(instance, remaining) {
		return 0, auctiontypes.InsufficientResources
	}

	//score first
	total := rep.delegate.TotalResources()
	nInstances := rep.delegate.NumInstancesForAppGuid(instance.AppGuid)
	score := rep.score(remaining, total, nInstances)

	//then reserve
	err := rep.delegate.Reserve(instance)
	if err != nil {
		return 0, err
	}

	return score, nil
}

func (rep *AuctionRep) ReleaseReservation(instance auctiontypes.LRPAuctionInfo) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.delegate.ReleaseReservation(instance)
}

func (rep *AuctionRep) Claim(instance models.LRPStartAuction) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.delegate.Claim(instance)
}

func (rep *AuctionRep) TotalResources() auctiontypes.Resources {
	return rep.delegate.TotalResources()
}

func (rep *AuctionRep) Reset() {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	simDelegate, ok := rep.delegate.(SimulationAuctionRepDelegate)
	if !ok {
		println("not reseting")
		return
	}
	simDelegate.SetLRPAuctionInfos([]auctiontypes.LRPAuctionInfo{})
}

func (rep *AuctionRep) SetLRPAuctionInfos(instances []auctiontypes.LRPAuctionInfo) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	simDelegate, ok := rep.delegate.(SimulationAuctionRepDelegate)
	if !ok {
		println("not setting instances")
		return
	}
	simDelegate.SetLRPAuctionInfos(instances)
}

func (rep *AuctionRep) LRPAuctionInfos() []auctiontypes.LRPAuctionInfo {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	simDelegate, ok := rep.delegate.(SimulationAuctionRepDelegate)
	if !ok {
		println("not fetching instances")
		return []auctiontypes.LRPAuctionInfo{}
	}
	return simDelegate.LRPAuctionInfos()
}

// internals -- no locks here the operations above should be atomic

func (rep *AuctionRep) hasRoomFor(instance auctiontypes.LRPAuctionInfo, remaining auctiontypes.Resources) bool {
	hasEnoughMemory := remaining.MemoryMB >= instance.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= instance.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	return hasEnoughMemory && hasEnoughDisk && hasEnoughContainers
}

func (rep *AuctionRep) score(remaining auctiontypes.Resources, total auctiontypes.Resources, nInstances int) float64 {
	fMemory := 1.0 - float64(remaining.MemoryMB)/float64(total.MemoryMB)
	fDisk := 1.0 - float64(remaining.DiskMB)/float64(total.DiskMB)
	fContainers := 1.0 - float64(remaining.Containers)/float64(total.Containers)
	fResources := (fMemory + fDisk + fContainers) / 3.0

	return fResources + float64(nInstances)
}
