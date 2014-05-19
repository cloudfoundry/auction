package simulationrepdelegate

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type SimulationRepDelegate struct {
	lock           *sync.Mutex
	instances      map[string]auctiontypes.LRPAuctionInfo
	totalResources auctiontypes.Resources
}

func New(totalResources auctiontypes.Resources) auctionrep.SimulationAuctionRepDelegate {
	return &SimulationRepDelegate{
		totalResources: totalResources,

		lock:      &sync.Mutex{},
		instances: map[string]auctiontypes.LRPAuctionInfo{},
	}
}

func (rep *SimulationRepDelegate) RemainingResources() auctiontypes.Resources {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.remainingResources()
}

func (rep *SimulationRepDelegate) TotalResources() auctiontypes.Resources {
	return rep.totalResources
}

func (rep *SimulationRepDelegate) NumInstancesForAppGuid(guid string) int {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	n := 0

	for _, instance := range rep.instances {
		if instance.AppGuid == guid {
			n += 1
		}
	}

	return n
}

func (rep *SimulationRepDelegate) Reserve(instance auctiontypes.LRPAuctionInfo) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.remainingResources()

	hasEnoughMemory := remaining.MemoryMB >= instance.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= instance.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	if !(hasEnoughMemory && hasEnoughDisk && hasEnoughContainers) {
		return auctiontypes.InsufficientResources
	}

	rep.instances[instance.InstanceGuid] = instance

	return nil
}

func (rep *SimulationRepDelegate) ReleaseReservation(instance auctiontypes.LRPAuctionInfo) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	reservedInstance, ok := rep.instances[instance.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", reservedInstance.InstanceGuid))
	}

	delete(rep.instances, instance.InstanceGuid)

	return nil
}

func (rep *SimulationRepDelegate) Claim(instance models.LRPStartAuction) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[instance.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", instance.InstanceGuid))
	}

	//start the app asynchronously!

	return nil
}

//simulation only

func (rep *SimulationRepDelegate) SetLRPAuctionInfos(instances []auctiontypes.LRPAuctionInfo) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	instancesMap := map[string]auctiontypes.LRPAuctionInfo{}
	for _, instance := range instances {
		instancesMap[instance.InstanceGuid] = instance
	}

	rep.instances = instancesMap
}

func (rep *SimulationRepDelegate) LRPAuctionInfos() []auctiontypes.LRPAuctionInfo {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	result := []auctiontypes.LRPAuctionInfo{}
	for _, instance := range rep.instances {
		result = append(result, instance)
	}
	return result
}

//internal

func (rep *SimulationRepDelegate) remainingResources() auctiontypes.Resources {
	resources := rep.totalResources
	for _, instance := range rep.instances {
		resources.MemoryMB -= instance.MemoryMB
		resources.DiskMB -= instance.DiskMB
		resources.Containers -= 1
	}
	return resources
}
