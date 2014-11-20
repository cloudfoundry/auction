package simulationrep

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type SimulationRep struct {
	stack          string
	totalResources auctiontypes.Resources
	instances      map[string]auctiontypes.LRP

	lock *sync.Mutex
}

func New(stack string, totalResources auctiontypes.Resources) auctiontypes.SimulationAuctionRep {
	return &SimulationRep{
		stack:          stack,
		totalResources: totalResources,
		instances:      map[string]auctiontypes.LRP{},

		lock: &sync.Mutex{},
	}
}

func (rep *SimulationRep) State() (auctiontypes.RepState, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	lrps := []auctiontypes.LRP{}
	for _, lrp := range rep.instances {
		lrps = append(lrps, lrp)
	}

	availableResources := rep.availableResources()

	// util.RandomSleep(800, 900)

	return auctiontypes.RepState{
		Stack:              rep.stack,
		AvailableResources: availableResources,
		TotalResources:     rep.totalResources,
		LRPs:               lrps,
	}, nil
}

func (rep *SimulationRep) Perform(work auctiontypes.Work) (auctiontypes.Work, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	failedWork := auctiontypes.Work{}

	for _, stop := range work.Stops {
		_, ok := rep.instances[stop.InstanceGuid]
		if !ok {
			failedWork.Stops = append(failedWork.Stops, stop)
			continue
		}
		delete(rep.instances, stop.InstanceGuid)
	}

	availableResources := rep.availableResources()

	for _, start := range work.Starts {
		hasRoom := availableResources.Containers > 0
		hasRoom = hasRoom && availableResources.MemoryMB > start.DesiredLRP.MemoryMB
		hasRoom = hasRoom && availableResources.DiskMB > start.DesiredLRP.DiskMB

		if hasRoom {
			rep.instances[start.InstanceGuid] = auctiontypes.LRP{
				ProcessGuid:  start.DesiredLRP.ProcessGuid,
				InstanceGuid: start.InstanceGuid,
				Index:        start.Index,
				MemoryMB:     start.DesiredLRP.MemoryMB,
				DiskMB:       start.DesiredLRP.DiskMB,
			}
			availableResources.Containers -= 1
			availableResources.MemoryMB -= start.DesiredLRP.MemoryMB
			availableResources.DiskMB -= start.DesiredLRP.DiskMB
		} else {
			failedWork.Starts = append(failedWork.Starts, start)
		}
	}

	// util.RandomSleep(800, 900)

	return failedWork, nil
}

//simulation only

func (rep *SimulationRep) Reset() error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	rep.instances = map[string]auctiontypes.LRP{}
	return nil
}

//internal -- no locks here

func (rep *SimulationRep) availableResources() auctiontypes.Resources {
	resources := rep.totalResources
	for _, instance := range rep.instances {
		resources.MemoryMB -= instance.MemoryMB
		resources.DiskMB -= instance.DiskMB
		resources.Containers -= 1
	}
	return resources
}
