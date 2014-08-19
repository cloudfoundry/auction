package auctionrep

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

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

	simDelegate, ok := rep.delegate.(auctiontypes.SimulationAuctionRepDelegate)
	if !ok {
		println("not reseting")
		return
	}
	simDelegate.SetSimulatedInstances([]auctiontypes.SimulatedInstance{})
}

// simulation-only
// must lock here; the publicly visible operations should be atomic
func (rep *AuctionRep) SetSimulatedInstances(instances []auctiontypes.SimulatedInstance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	simDelegate, ok := rep.delegate.(auctiontypes.SimulationAuctionRepDelegate)
	if !ok {
		println("not setting instances")
		return
	}
	simDelegate.SetSimulatedInstances(instances)
}

// simulation-only
// must lock here; the publicly visible operations should be atomic
func (rep *AuctionRep) SimulatedInstances() []auctiontypes.SimulatedInstance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	simDelegate, ok := rep.delegate.(auctiontypes.SimulationAuctionRepDelegate)
	if !ok {
		println("not fetching instances")
		return []auctiontypes.SimulatedInstance{}
	}
	return simDelegate.SimulatedInstances()
}
