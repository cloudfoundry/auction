package simulationrep

import (
	"sync"

	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/rep"
)

type SimulationRep struct {
	stack          string
	zone           string
	totalResources rep.Resources
	lrps           map[string]rep.LRP
	tasks          map[string]rep.Task

	lock *sync.Mutex
}

func New(stack string, zone string, totalResources rep.Resources) rep.SimClient {
	return &SimulationRep{
		stack:          stack,
		totalResources: totalResources,
		lrps:           map[string]rep.LRP{},
		tasks:          map[string]rep.Task{},
		zone:           zone,

		lock: &sync.Mutex{},
	}
}

func (r *SimulationRep) State() (rep.CellState, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	lrps := []rep.LRP{}
	for _, lrp := range r.lrps {
		lrps = append(lrps, lrp)
	}

	tasks := []rep.Task{}
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}

	availableResources := r.availableResources()

	// util.RandomSleep(800, 900)

	return rep.CellState{
		RootFSProviders: rep.RootFSProviders{
			models.PreloadedRootFSScheme: rep.NewFixedSetRootFSProvider(r.stack),
		},
		AvailableResources: availableResources,
		TotalResources:     r.totalResources,
		LRPs:               lrps,
		Tasks:              tasks,
		Zone:               r.zone,
	}, nil
}

func (r *SimulationRep) Perform(work rep.Work) (rep.Work, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	failedWork := rep.Work{}

	availableResources := r.availableResources()

	for _, start := range work.LRPs {

		hasRoom := availableResources.Containers >= 0
		hasRoom = hasRoom && availableResources.MemoryMB >= start.MemoryMB
		hasRoom = hasRoom && availableResources.DiskMB >= start.DiskMB

		if hasRoom {
			r.lrps[start.Identifier()] = start

			availableResources.Containers -= 1
			availableResources.MemoryMB -= start.MemoryMB
			availableResources.DiskMB -= start.DiskMB
		} else {
			failedWork.LRPs = append(failedWork.LRPs, start)
		}
	}

	for _, task := range work.Tasks {
		hasRoom := availableResources.Containers >= 0
		hasRoom = hasRoom && availableResources.MemoryMB >= task.MemoryMB
		hasRoom = hasRoom && availableResources.DiskMB >= task.DiskMB

		if hasRoom {
			r.tasks[task.TaskGuid] = task

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

func (r *SimulationRep) Reset() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.lrps = map[string]rep.LRP{}
	r.tasks = map[string]rep.Task{}
	return nil
}

//these are rep client methods the auction does not use

func (rep *SimulationRep) StopLRPInstance(models.ActualLRPKey, models.ActualLRPInstanceKey) error {
	panic("UNIMPLEMENTED METHOD")
}

func (rep *SimulationRep) CancelTask(string) error {
	panic("UNIMPLEMENTED METHOD")
}

//internal -- no locks here

func (rep *SimulationRep) availableResources() rep.Resources {
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
