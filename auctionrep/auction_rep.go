package auctionrep

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type AuctionRepDelegate interface {
	RemainingResources() (auctiontypes.Resources, error)
	TotalResources() (auctiontypes.Resources, error)
	NumInstancesForAppGuid(guid string) (int, error)

	Reserve(instance auctiontypes.LRPAuctionInfo) error
	ReleaseReservation(instance auctiontypes.LRPAuctionInfo) error
	Run(instance models.LRPStartAuction) error
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

type RepInstanceScoreInfo struct {
	RemainingResources     auctiontypes.Resources
	TotalResources         auctiontypes.Resources
	NumInstancesForAppGuid int
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

// must lock here; the publicly visible operations should be atomic
func (rep *AuctionRep) Score(instance auctiontypes.LRPAuctionInfo) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	repInstanceScoreInfo, err := rep.repInstanceScoreInfo(instance)
	if err != nil {
		return 0, err
	}

	err = rep.satisfiesConstraints(instance, repInstanceScoreInfo)
	if err != nil {
		return 0, err
	}

	return rep.score(repInstanceScoreInfo), nil
}

// must lock here; the publicly visible operations should be atomic
func (rep *AuctionRep) ScoreThenTentativelyReserve(instance auctiontypes.LRPAuctionInfo) (float64, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	repInstanceScoreInfo, err := rep.repInstanceScoreInfo(instance)
	if err != nil {
		return 0, err
	}

	err = rep.satisfiesConstraints(instance, repInstanceScoreInfo)
	if err != nil {
		return 0, err
	}

	score := rep.score(repInstanceScoreInfo)

	//then reserve
	err = rep.delegate.Reserve(instance)
	if err != nil {
		return 0, err
	}

	return score, nil
}

// must lock here; the publicly visible operations should be atomic
func (rep *AuctionRep) ReleaseReservation(instance auctiontypes.LRPAuctionInfo) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.delegate.ReleaseReservation(instance)
}

// must lock here; the publicly visible operations should be atomic
func (rep *AuctionRep) Run(instance models.LRPStartAuction) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.delegate.Run(instance)
}

// simulation-only
func (rep *AuctionRep) TotalResources() auctiontypes.Resources {
	totalResources, _ := rep.delegate.TotalResources()
	return totalResources
}

// simulation-only
// must lock here; the publicly visible operations should be atomic
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

// simulation-only
// must lock here; the publicly visible operations should be atomic
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

// simulation-only
// must lock here; the publicly visible operations should be atomic
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

// private internals -- no locks here
func (rep *AuctionRep) repInstanceScoreInfo(instance auctiontypes.LRPAuctionInfo) (RepInstanceScoreInfo, error) {
	remaining, err := rep.delegate.RemainingResources()
	if err != nil {
		return RepInstanceScoreInfo{}, err
	}

	total, err := rep.delegate.TotalResources()
	if err != nil {
		return RepInstanceScoreInfo{}, err
	}

	nInstances, err := rep.delegate.NumInstancesForAppGuid(instance.AppGuid)
	if err != nil {
		return RepInstanceScoreInfo{}, err
	}

	return RepInstanceScoreInfo{
		RemainingResources:     remaining,
		TotalResources:         total,
		NumInstancesForAppGuid: nInstances,
	}, nil
}

// private internals -- no locks here
func (rep *AuctionRep) satisfiesConstraints(instance auctiontypes.LRPAuctionInfo, repInstanceScoreInfo RepInstanceScoreInfo) error {
	remaining := repInstanceScoreInfo.RemainingResources
	hasEnoughMemory := remaining.MemoryMB >= instance.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= instance.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	if hasEnoughMemory && hasEnoughDisk && hasEnoughContainers {
		return nil
	} else {
		return auctiontypes.InsufficientResources
	}
}

// private internals -- no locks here
func (rep *AuctionRep) score(repInstanceScoreInfo RepInstanceScoreInfo) float64 {
	remaining := repInstanceScoreInfo.RemainingResources
	total := repInstanceScoreInfo.TotalResources

	fractionUsedContainers := 1.0 - float64(remaining.Containers)/float64(total.Containers)
	fractionUsedDisk := 1.0 - float64(remaining.DiskMB)/float64(total.DiskMB)
	fractionUsedMemory := 1.0 - float64(remaining.MemoryMB)/float64(total.MemoryMB)

	return ((fractionUsedContainers + fractionUsedDisk + fractionUsedMemory) / 3.0) + float64(repInstanceScoreInfo.NumInstancesForAppGuid)
}
