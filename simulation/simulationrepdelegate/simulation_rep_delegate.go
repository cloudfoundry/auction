package simulationrepdelegate

import (
	"errors"
	"fmt"
	"sync"

	"github.com/onsi/auction/auctionrep"
	"github.com/onsi/auction/types"
)

type SimulationRepDelegate struct {
	lock           *sync.Mutex
	instances      map[string]types.Instance
	totalResources types.Resources
}

func New(totalResources types.Resources) auctionrep.SimulationAuctionRepDelegate {
	return &SimulationRepDelegate{
		totalResources: totalResources,

		lock:      &sync.Mutex{},
		instances: map[string]types.Instance{},
	}
}

func (rep *SimulationRepDelegate) RemainingResources() types.Resources {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	return rep.remainingResources()
}

func (rep *SimulationRepDelegate) TotalResources() types.Resources {
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

func (rep *SimulationRepDelegate) Reserve(instance types.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.remainingResources()

	hasEnoughMemory := remaining.MemoryMB >= instance.Resources.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= instance.Resources.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	if !(hasEnoughMemory && hasEnoughDisk && hasEnoughContainers) {
		return types.InsufficientResources
	}

	rep.instances[instance.InstanceGuid] = instance

	return nil
}

func (rep *SimulationRepDelegate) ReleaseReservation(instance types.Instance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	reservedInstance, ok := rep.instances[instance.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", reservedInstance.InstanceGuid))
	}

	delete(rep.instances, instance.InstanceGuid)

	return nil
}

func (rep *SimulationRepDelegate) Claim(instance types.Instance) error {
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

func (rep *SimulationRepDelegate) SetInstances(instances []types.Instance) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	instancesMap := map[string]types.Instance{}
	for _, instance := range instances {
		instancesMap[instance.InstanceGuid] = instance
	}

	rep.instances = instancesMap
}

func (rep *SimulationRepDelegate) Instances() []types.Instance {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	result := []types.Instance{}
	for _, instance := range rep.instances {
		result = append(result, instance)
	}
	return result
}

//internal

func (rep *SimulationRepDelegate) remainingResources() types.Resources {
	resources := rep.totalResources
	for _, instance := range rep.instances {
		resources.MemoryMB -= instance.Resources.MemoryMB
		resources.DiskMB -= instance.Resources.DiskMB
		resources.Containers -= 1
	}
	return resources
}
