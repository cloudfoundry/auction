package auctionrep

import (
	"sync"

	"github.com/onsi/auction/types"
)

type AuctionRepDelegate interface {
	RemainingResources() types.Resources
	TotalResources() types.Resources

	NumInstancesForAppGuid(guid string) int
	Reserve(instance types.Instance) error
	ReleaseReservation(instance types.Instance) error
	Claim(instance types.Instance) error
}

//Used in simulation
type SimulationAuctionRepDelegate interface {
	AuctionRepDelegate
	SetInstances(instances []types.Instance)
	Instances() []types.Instance
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

func (rep *AuctionRep) Score(instance types.Instance) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.delegate.RemainingResources()
	if !rep.hasRoomFor(instance.Resources, remaining) {
		return 0, types.InsufficientResources
	}

	total := rep.delegate.TotalResources()
	nInstances := rep.delegate.NumInstancesForAppGuid(instance.AppGuid)

	return rep.score(remaining, total, nInstances), nil
}

func (rep *AuctionRep) ScoreThenTentativelyReserve(instance types.Instance) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.delegate.RemainingResources()
	if !rep.hasRoomFor(instance.Resources, remaining) {
		return 0, types.InsufficientResources
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

func (rep *AuctionRep) ReleaseReservation(instance types.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.delegate.ReleaseReservation(instance)
}

func (rep *AuctionRep) Claim(instance types.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.delegate.Claim(instance)
}

func (rep *AuctionRep) TotalResources() types.Resources {
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
	simDelegate.SetInstances([]types.Instance{})
}

func (rep *AuctionRep) SetInstances(instances []types.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	simDelegate, ok := rep.delegate.(SimulationAuctionRepDelegate)
	if !ok {
		println("not setting instances")
		return
	}
	simDelegate.SetInstances(instances)
}

func (rep *AuctionRep) Instances() []types.Instance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	simDelegate, ok := rep.delegate.(SimulationAuctionRepDelegate)
	if !ok {
		println("not fetching instances")
		return []types.Instance{}
	}
	return simDelegate.Instances()
}

// internals -- no locks here the operations above should be atomic

func (rep *AuctionRep) hasRoomFor(required types.Resources, remaining types.Resources) bool {
	hasEnoughMemory := remaining.MemoryMB >= required.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= required.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	return hasEnoughMemory && hasEnoughDisk && hasEnoughContainers
}

func (rep *AuctionRep) score(remaining types.Resources, total types.Resources, nInstances int) float64 {
	fMemory := 1.0 - remaining.MemoryMB/total.MemoryMB
	fDisk := 1.0 - remaining.DiskMB/total.DiskMB
	fContainers := 1.0 - float64(remaining.Containers)/float64(total.Containers)
	fResources := (fMemory + fDisk + fContainers) / 3.0

	return fResources + float64(nInstances)
}
