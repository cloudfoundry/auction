package auctionrunner

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type Cell struct {
	Guid   string
	client auctiontypes.CellRep
	state  auctiontypes.CellState

	workToCommit auctiontypes.Work
}

func NewCell(guid string, client auctiontypes.CellRep, state auctiontypes.CellState) *Cell {
	return &Cell{
		Guid:   guid,
		client: client,
		state:  state,
	}
}

func (c *Cell) Stack() string {
	return c.state.Stack
}

func (c *Cell) ScoreForLRPAuction(lrpAuction auctiontypes.LRPAuction) (float64, error) {
	err := c.canHandleLRPAuction(lrpAuction)
	if err != nil {
		return 0, err
	}

	numberOfInstancesWithMatchingProcessGuid := 0
	for _, lrp := range c.state.LRPs {
		if lrp.ProcessGuid == lrpAuction.DesiredLRP.ProcessGuid {
			numberOfInstancesWithMatchingProcessGuid++
		}
	}

	remainingResources := c.state.AvailableResources
	remainingResources.MemoryMB -= lrpAuction.DesiredLRP.MemoryMB
	remainingResources.DiskMB -= lrpAuction.DesiredLRP.DiskMB
	remainingResources.Containers -= 1

	resourceScore := c.computeScore(remainingResources, numberOfInstancesWithMatchingProcessGuid)

	return resourceScore, nil
}

func (c *Cell) ScoreForTask(task models.Task) (float64, error) {
	err := c.canHandleTask(task)
	if err != nil {
		return 0, err
	}

	remainingResources := c.state.AvailableResources
	remainingResources.MemoryMB -= task.MemoryMB
	remainingResources.DiskMB -= task.DiskMB
	remainingResources.Containers -= 1

	resourceScore := c.computeTaskScore(remainingResources)

	return resourceScore, nil
}

func (c *Cell) ReserveLRP(lrpAuction auctiontypes.LRPAuction) error {
	err := c.canHandleLRPAuction(lrpAuction)
	if err != nil {
		return err
	}

	c.state.LRPs = append(c.state.LRPs, auctiontypes.LRP{
		ProcessGuid: lrpAuction.DesiredLRP.ProcessGuid,
		Index:       lrpAuction.Index,
		MemoryMB:    lrpAuction.DesiredLRP.MemoryMB,
		DiskMB:      lrpAuction.DesiredLRP.DiskMB,
	})

	c.state.AvailableResources.MemoryMB -= lrpAuction.DesiredLRP.MemoryMB
	c.state.AvailableResources.DiskMB -= lrpAuction.DesiredLRP.DiskMB
	c.state.AvailableResources.Containers -= 1

	c.workToCommit.LRPs = append(c.workToCommit.LRPs, lrpAuction)

	return nil
}

func (c *Cell) ReserveTask(task models.Task) error {
	err := c.canHandleTask(task)
	if err != nil {
		return err
	}

	c.state.Tasks = append(c.state.Tasks, auctiontypes.Task{
		TaskGuid: task.TaskGuid,
		MemoryMB: task.MemoryMB,
		DiskMB:   task.DiskMB,
	})

	c.state.AvailableResources.MemoryMB -= task.MemoryMB
	c.state.AvailableResources.DiskMB -= task.DiskMB
	c.state.AvailableResources.Containers -= 1

	c.workToCommit.Tasks = append(c.workToCommit.Tasks, task)

	return nil
}

func (c *Cell) Commit() auctiontypes.Work {
	if len(c.workToCommit.LRPs) == 0 && len(c.workToCommit.Tasks) == 0 {
		return auctiontypes.Work{}
	}

	failedWork, err := c.client.Perform(c.workToCommit)
	if err != nil {
		//an error may indicate partial failure
		//in this case we don't reschedule work in order to make sure we don't
		//create duplicates of things -- we'll let the converger figure things out for us later
		return auctiontypes.Work{}
	}
	return failedWork
}

func (c *Cell) canHandleLRPAuction(lrpAuction auctiontypes.LRPAuction) error {
	if c.state.Stack != lrpAuction.DesiredLRP.Stack {
		return auctiontypes.ErrorCellMismatch
	}
	if c.state.AvailableResources.MemoryMB < lrpAuction.DesiredLRP.MemoryMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.DiskMB < lrpAuction.DesiredLRP.DiskMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.Containers < 1 {
		return auctiontypes.ErrorInsufficientResources
	}

	return nil
}

func (c *Cell) canHandleTask(task models.Task) error {
	if c.state.Stack != task.Stack {
		return auctiontypes.ErrorCellMismatch
	}
	if c.state.AvailableResources.MemoryMB < task.MemoryMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.DiskMB < task.DiskMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.Containers < 1 {
		return auctiontypes.ErrorInsufficientResources
	}

	return nil
}

func (c *Cell) computeScore(remainingResources auctiontypes.Resources, numInstances int) float64 {
	fractionUsedMemory := 1.0 - float64(remainingResources.MemoryMB)/float64(c.state.TotalResources.MemoryMB)
	fractionUsedDisk := 1.0 - float64(remainingResources.DiskMB)/float64(c.state.TotalResources.DiskMB)
	fractionUsedContainers := 1.0 - float64(remainingResources.Containers)/float64(c.state.TotalResources.Containers)

	resourceScore := (fractionUsedMemory + fractionUsedDisk + fractionUsedContainers) / 3.0
	resourceScore += float64(numInstances)

	return resourceScore
}

func (c *Cell) computeTaskScore(remainingResources auctiontypes.Resources) float64 {
	fractionUsedMemory := 1.0 - float64(remainingResources.MemoryMB)/float64(c.state.TotalResources.MemoryMB)
	fractionUsedDisk := 1.0 - float64(remainingResources.DiskMB)/float64(c.state.TotalResources.DiskMB)
	fractionUsedContainers := 1.0 - float64(remainingResources.Containers)/float64(c.state.TotalResources.Containers)

	resourceScore := (fractionUsedMemory + fractionUsedDisk + fractionUsedContainers) / 3.0

	return resourceScore
}
