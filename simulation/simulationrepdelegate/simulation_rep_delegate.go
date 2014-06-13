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
	instances      map[string]auctiontypes.SimulatedInstance
	totalResources auctiontypes.Resources
}

func New(totalResources auctiontypes.Resources) auctionrep.SimulationAuctionRepDelegate {
	return &SimulationRepDelegate{
		totalResources: totalResources,

		lock:      &sync.Mutex{},
		instances: map[string]auctiontypes.SimulatedInstance{},
	}
}

func (rep *SimulationRepDelegate) RemainingResources() (auctiontypes.Resources, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.remainingResources(), nil
}

func (rep *SimulationRepDelegate) TotalResources() (auctiontypes.Resources, error) {
	return rep.totalResources, nil
}

func (rep *SimulationRepDelegate) NumInstancesForAppGuid(guid string) (int, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	n := 0

	for _, instance := range rep.instances {
		if instance.ProcessGuid == guid {
			n += 1
		}
	}

	return n, nil
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

	rep.instances[instance.InstanceGuid] = auctiontypes.SimulatedInstance{
		ProcessGuid:  instance.AppGuid,
		InstanceGuid: instance.InstanceGuid,
		MemoryMB:     instance.MemoryMB,
		DiskMB:       instance.DiskMB,
	}

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

func (rep *SimulationRepDelegate) Run(instance models.LRPStartAuction) error {
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

func (rep *SimulationRepDelegate) SetSimulatedInstances(instances []auctiontypes.SimulatedInstance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	instancesMap := map[string]auctiontypes.SimulatedInstance{}
	for _, instance := range instances {
		instancesMap[instance.InstanceGuid] = instance
	}

	rep.instances = instancesMap
}

func (rep *SimulationRepDelegate) SimulatedInstances() []auctiontypes.SimulatedInstance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	result := []auctiontypes.SimulatedInstance{}
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
