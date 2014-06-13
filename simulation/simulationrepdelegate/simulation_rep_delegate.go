package simulationrepdelegate

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type SimulationRepDelegate struct {
	lock           *sync.Mutex
	instances      map[string]auctiontypes.SimulatedInstance
	totalResources auctiontypes.Resources
}

func New(totalResources auctiontypes.Resources) auctiontypes.SimulationAuctionRepDelegate {
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

func (rep *SimulationRepDelegate) InstanceGuidsForProcessGuidAndIndex(processGuid string, index int) ([]string, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	instanceGuids := []string{}

	for _, instance := range rep.instances {
		if instance.ProcessGuid == processGuid && instance.Index == index {
			instanceGuids = append(instanceGuids, instance.InstanceGuid)
		}
	}

	return instanceGuids, nil
}

func (rep *SimulationRepDelegate) Reserve(startAuctionInfo auctiontypes.LRPStartAuctionInfo) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.remainingResources()

	hasEnoughMemory := remaining.MemoryMB >= startAuctionInfo.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= startAuctionInfo.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	if !(hasEnoughMemory && hasEnoughDisk && hasEnoughContainers) {
		return auctiontypes.InsufficientResources
	}

	rep.instances[startAuctionInfo.InstanceGuid] = auctiontypes.SimulatedInstance{
		ProcessGuid:  startAuctionInfo.AppGuid,
		InstanceGuid: startAuctionInfo.InstanceGuid,
		MemoryMB:     startAuctionInfo.MemoryMB,
		DiskMB:       startAuctionInfo.DiskMB,
	}

	return nil
}

func (rep *SimulationRepDelegate) ReleaseReservation(startAuctionInfo auctiontypes.LRPStartAuctionInfo) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[startAuctionInfo.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", startAuctionInfo.InstanceGuid))
	}

	delete(rep.instances, startAuctionInfo.InstanceGuid)

	return nil
}

func (rep *SimulationRepDelegate) Run(startAuction models.LRPStartAuction) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[startAuction.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", startAuction.InstanceGuid))
	}

	//start the app asynchronously!

	return nil
}

func (rep *SimulationRepDelegate) Stop(instanceGuid string) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[instanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", instanceGuid))
	}

	delete(rep.instances, instanceGuid)

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
