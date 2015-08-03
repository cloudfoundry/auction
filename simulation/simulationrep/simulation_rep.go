package simulationrep

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/bbs/models"
)

type SimulationRep struct {
	stack          string
	zone           string
	totalResources auctiontypes.Resources
	lrps           map[string]auctiontypes.LRP
	tasks          map[string]auctiontypes.Task

	lock *sync.Mutex
}

func New(stack string, zone string, totalResources auctiontypes.Resources) auctiontypes.SimulationCellRep {
	return &SimulationRep{
		stack:          stack,
		totalResources: totalResources,
		lrps:           map[string]auctiontypes.LRP{},
		tasks:          map[string]auctiontypes.Task{},
		zone:           zone,

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
		RootFSProviders: auctiontypes.RootFSProviders{
			models.PreloadedRootFSScheme: auctiontypes.NewFixedSetRootFSProvider(rep.stack),
		},
		AvailableResources: availableResources,
		TotalResources:     rep.totalResources,
		LRPs:               lrps,
		Tasks:              tasks,
		Zone:               rep.zone,
	}, nil
}

func (rep *SimulationRep) Perform(work auctiontypes.Work) (auctiontypes.Work, error) {
	rep.lock.Lock()
	defer rep.lock.Unlock()

	failedWork := auctiontypes.Work{}

	availableResources := rep.availableResources()

	for _, start := range work.LRPs {
		hasRoom := availableResources.Containers >= 0
		hasRoom = hasRoom && availableResources.MemoryMB >= int(start.DesiredLRP.MemoryMb)
		hasRoom = hasRoom && availableResources.DiskMB >= int(start.DesiredLRP.DiskMb)

		if hasRoom {
			rep.lrps[auctiontypes.IdentifierForLRP(start.DesiredLRP.ProcessGuid, start.Index)] = auctiontypes.LRP{
				ProcessGuid: start.DesiredLRP.ProcessGuid,
				Index:       start.Index,
				MemoryMB:    int(start.DesiredLRP.MemoryMb),
				DiskMB:      int(start.DesiredLRP.DiskMb),
			}
			availableResources.Containers -= 1
			availableResources.MemoryMB -= int(start.DesiredLRP.MemoryMb)
			availableResources.DiskMB -= int(start.DesiredLRP.DiskMb)
		} else {
			failedWork.LRPs = append(failedWork.LRPs, start)
		}
	}

	for _, task := range work.Tasks {
		hasRoom := availableResources.Containers >= 0
		hasRoom = hasRoom && availableResources.MemoryMB >= int(task.MemoryMb)
		hasRoom = hasRoom && availableResources.DiskMB >= int(task.DiskMb)

		if hasRoom {
			rep.tasks[task.TaskGuid] = auctiontypes.Task{
				TaskGuid: task.TaskGuid,
				MemoryMB: int(task.MemoryMb),
				DiskMB:   int(task.DiskMb),
			}
			availableResources.Containers -= 1
			availableResources.MemoryMB -= int(task.MemoryMb)
			availableResources.DiskMB -= int(task.DiskMb)
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
