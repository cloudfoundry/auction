package simulationrepdelegate

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/util"
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

func (rep *SimulationRepDelegate) NumInstancesForProcessGuid(processGuid string) (int, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	n := 0

	for _, instance := range rep.instances {
		if instance.ProcessGuid == processGuid {
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

func (rep *SimulationRepDelegate) Reserve(startAuction models.LRPStartAuction) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	remaining := rep.remainingResources()

	hasEnoughMemory := remaining.MemoryMB >= startAuction.DesiredLRP.MemoryMB
	hasEnoughDisk := remaining.DiskMB >= startAuction.DesiredLRP.DiskMB
	hasEnoughContainers := remaining.Containers > 0

	if !(hasEnoughMemory && hasEnoughDisk && hasEnoughContainers) {
		return auctiontypes.InsufficientResources
	}

	rep.instances[startAuction.InstanceGuid] = auctiontypes.SimulatedInstance{
		ProcessGuid:  startAuction.DesiredLRP.ProcessGuid,
		InstanceGuid: startAuction.InstanceGuid,
		MemoryMB:     startAuction.DesiredLRP.MemoryMB,
		DiskMB:       startAuction.DesiredLRP.DiskMB,
		Index:        startAuction.Index,
	}

	return nil
}

func (rep *SimulationRepDelegate) ReleaseReservation(startAuction models.LRPStartAuction) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[startAuction.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", startAuction.InstanceGuid))
	}

	delete(rep.instances, startAuction.InstanceGuid)

	return nil
}

func (rep *SimulationRepDelegate) Run(startAuction models.LRPStartAuction) error {
	rep.lock.Lock()
	_, ok := rep.instances[startAuction.InstanceGuid]
	rep.lock.Unlock()

	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", startAuction.InstanceGuid))
	}

	util.RandomSleep(800, 900)

	return nil
}

func (rep *SimulationRepDelegate) Stop(stopInstance models.StopLRPInstance) error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	_, ok := rep.instances[stopInstance.InstanceGuid]
	if !ok {
		return errors.New(fmt.Sprintf("no reservation for instance %s", stopInstance.InstanceGuid))
	}

	delete(rep.instances, stopInstance.InstanceGuid)

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
