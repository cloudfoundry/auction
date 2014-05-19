package simulationrepdelegate

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type SimulationRepDelegate struct {
	lock           *sync.Mutex
	instances      map[string]auctiontypes.Instance
	totalResources auctiontypes.Resources
}

func New(totalResources auctiontypes.Resources) auctionrep.SimulationAuctionRepDelegate {
	return &SimulationRepDelegate{
		totalResources: totalResources,

		lock:      &sync.Mutex{},
		instances: map[string]auctiontypes.Instance{},
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

func (rep *SimulationRepDelegate) Reserve(instance auctiontypes.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.remainingResources()

	hasEnoughMemory := remaining.MemoryMB >= instance.Resources.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= instance.Resources.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	if !(hasEnoughMemory && hasEnoughDisk && hasEnoughContainers) {
		return auctiontypes.InsufficientResources
	}

	rep.instances[instance.InstanceGuid] = instance

	return nil
}

func (rep *SimulationRepDelegate) ReleaseReservation(instance auctiontypes.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	reservedInstance, ok := rep.instances[instance.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", reservedInstance.InstanceGuid))
	}

	delete(rep.instances, instance.InstanceGuid)

	return nil
}

func (rep *SimulationRepDelegate) Claim(instance auctiontypes.Instance) error {
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

func (rep *SimulationRepDelegate) SetInstances(instances []auctiontypes.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	instancesMap := map[string]auctiontypes.Instance{}
	for _, instance := range instances {
		instancesMap[instance.InstanceGuid] = instance
	}

	rep.instances = instancesMap
}

func (rep *SimulationRepDelegate) Instances() []auctiontypes.Instance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	result := []auctiontypes.Instance{}
	for _, instance := range rep.instances {
		result = append(result, instance)
	}
	return result
}

//internal

func (rep *SimulationRepDelegate) remainingResources() auctiontypes.Resources {
	resources := rep.totalResources
	for _, instance := range rep.instances {
		resources.MemoryMB -= instance.Resources.MemoryMB
		resources.DiskMB -= instance.Resources.DiskMB
		resources.Containers -= 1
	}
	return resources
}
