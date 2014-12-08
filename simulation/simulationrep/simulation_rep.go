package simulationrep

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type SimulationRep struct {
	stack          string
	totalResources auctiontypes.Resources
	lrps           map[string]auctiontypes.LRP
	tasks          map[string]auctiontypes.Task

	lock *sync.Mutex
}

func New(stack string, totalResources auctiontypes.Resources) auctiontypes.SimulationCellRep {
	return &SimulationRep{
		stack:          stack,
		totalResources: totalResources,
		lrps:           map[string]auctiontypes.LRP{},
		tasks:          map[string]auctiontypes.Task{},

		lock: &sync.Mutex{},
	}
}

func (rep *SimulationRep) State() (auctiontypes.CellState, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	lrps := []auctiontypes.LRP{}
	for _, lrp := range rep.lrps {
		lrps = append(lrps, lrp)
	}

	tasks := []auctiontypes.Task{}
	for _, task := range rep.tasks {
		tasks = append(tasks, task)
	}

	availableResources := rep.availableResources()

	// util.RandomSleep(800, 900)

	return auctiontypes.CellState{
		Stack:              rep.stack,
		AvailableResources: availableResources,
		TotalResources:     rep.totalResources,
		LRPs:               lrps,
		Tasks:              tasks,
	}, nil
}

func (rep *SimulationRep) Perform(work auctiontypes.Work) (auctiontypes.Work, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	failedWork := auctiontypes.Work{}

	for _, stop := range work.LRPStops {
		_, ok := rep.lrps[stop.InstanceGuid]
		if !ok {
			failedWork.LRPStops = append(failedWork.LRPStops, stop)
			continue
		}
		delete(rep.lrps, stop.InstanceGuid)
	}

	availableResources := rep.availableResources()

	for _, start := range work.LRPStarts {
		hasRoom := availableResources.Containers >= 0
		hasRoom = hasRoom && availableResources.MemoryMB >= start.DesiredLRP.MemoryMB
		hasRoom = hasRoom && availableResources.DiskMB >= start.DesiredLRP.DiskMB

		if hasRoom {
			rep.lrps[start.InstanceGuid] = auctiontypes.LRP{
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
			failedWork.LRPStarts = append(failedWork.LRPStarts, start)
		}
	}

	for _, task := range work.Tasks {
		hasRoom := availableResources.Containers >= 0
		hasRoom = hasRoom && availableResources.MemoryMB >= task.MemoryMB
		hasRoom = hasRoom && availableResources.DiskMB >= task.DiskMB

		if hasRoom {
			rep.tasks[task.TaskGuid] = auctiontypes.Task{
				TaskGuid: task.TaskGuid,
				MemoryMB: task.MemoryMB,
				DiskMB:   task.DiskMB,
			}
			availableResources.Containers -= 1
			availableResources.MemoryMB -= task.MemoryMB
			availableResources.DiskMB -= task.DiskMB
		} else {
			failedWork.Tasks = append(failedWork.Tasks, task)
		}
	}

	return failedWork, nil
}

//simulation only

func (rep *SimulationRep) Reset() error {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	rep.lrps = map[string]auctiontypes.LRP{}
	rep.tasks = map[string]auctiontypes.Task{}
	return nil
}

//internal -- no locks here

func (rep *SimulationRep) availableResources() auctiontypes.Resources {
	resources := rep.totalResources
	for _, lrp := range rep.lrps {
		resources.MemoryMB -= lrp.MemoryMB
		resources.DiskMB -= lrp.DiskMB
		resources.Containers -= 1
	}
	for _, task := range rep.tasks {
		resources.MemoryMB -= task.MemoryMB
		resources.DiskMB -= task.DiskMB
		resources.Containers -= 1
	}
	return resources
}
