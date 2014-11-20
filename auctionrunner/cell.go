package auctionrunner

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type Cell struct {
	client auctiontypes.AuctionRep
	state  auctiontypes.RepState

	workToCommit auctiontypes.Work
}

func NewCell(client auctiontypes.AuctionRep, state auctiontypes.RepState) *Cell {
	return &Cell{
		client: client,
		state:  state,
	}
}

func (c *Cell) ScoreForStartAuction(startAuction models.LRPStartAuction) (float64, error) {
	err := c.canHandleStartAuction(startAuction)
	if err != nil {
		return 0, err
	}

	numberOfInstancesWithMatchingProcessGuid := 0
	for _, lrp := range c.state.LRPs {
		if lrp.ProcessGuid == startAuction.DesiredLRP.ProcessGuid {
			numberOfInstancesWithMatchingProcessGuid++
		}
	}

	remainingResources := c.state.AvailableResources
	remainingResources.MemoryMB -= startAuction.DesiredLRP.MemoryMB
	remainingResources.DiskMB -= startAuction.DesiredLRP.DiskMB
	remainingResources.Containers -= 1

	resourceScore := c.computeScore(remainingResources, numberOfInstancesWithMatchingProcessGuid)

	return resourceScore, nil
}

func (c *Cell) ScoreForStopAuction(stopAuction models.LRPStopAuction) (float64, []string, error) {
	matchingLRPs := []auctiontypes.LRP{}
	numberOfInstancesWithMatchingProcessGuidButDifferentIndex := 0
	for _, lrp := range c.state.LRPs {
		if lrp.ProcessGuid == stopAuction.ProcessGuid {
			if lrp.Index == stopAuction.Index {
				matchingLRPs = append(matchingLRPs, lrp)
			} else {
				numberOfInstancesWithMatchingProcessGuidButDifferentIndex++
			}
		}
	}

	if len(matchingLRPs) == 0 {
		return 0, nil, auctiontypes.ErrorNothingToStop
	}

	remainingResources := c.state.AvailableResources
	instanceGuids := make([]string, len(matchingLRPs))

	for i, lrp := range matchingLRPs {
		instanceGuids[i] = lrp.InstanceGuid
		remainingResources.MemoryMB += lrp.MemoryMB
		remainingResources.DiskMB += lrp.DiskMB
		remainingResources.Containers += 1
	}

	resourceScore := c.computeScore(remainingResources, numberOfInstancesWithMatchingProcessGuidButDifferentIndex)

	return resourceScore, instanceGuids, nil
}

func (c *Cell) StartLRP(startAuction models.LRPStartAuction) error {
	err := c.canHandleStartAuction(startAuction)
	if err != nil {
		return err
	}

	c.state.LRPs = append(c.state.LRPs, auctiontypes.LRP{
		ProcessGuid:  startAuction.DesiredLRP.ProcessGuid,
		InstanceGuid: startAuction.InstanceGuid,
		Index:        startAuction.Index,
		MemoryMB:     startAuction.DesiredLRP.MemoryMB,
		DiskMB:       startAuction.DesiredLRP.DiskMB,
	})

	c.state.AvailableResources.MemoryMB -= startAuction.DesiredLRP.MemoryMB
	c.state.AvailableResources.DiskMB -= startAuction.DesiredLRP.DiskMB
	c.state.AvailableResources.Containers -= 1

	c.workToCommit.Starts = append(c.workToCommit.Starts, startAuction)

	return nil
}

func (c *Cell) StopLRP(stop models.StopLRPInstance) error {
	indexToDelete := -1
	for i, lrp := range c.state.LRPs {
		if lrp.ProcessGuid != stop.ProcessGuid {
			continue
		}
		if lrp.InstanceGuid != stop.InstanceGuid {
			continue
		}
		if lrp.Index != stop.Index {
			continue
		}
		indexToDelete = i
		break
	}

	if indexToDelete == -1 {
		return auctiontypes.ErrorNothingToStop
	}

	c.state.AvailableResources.MemoryMB += c.state.LRPs[indexToDelete].MemoryMB
	c.state.AvailableResources.DiskMB += c.state.LRPs[indexToDelete].DiskMB
	c.state.AvailableResources.Containers += 1

	c.state.LRPs = append(c.state.LRPs[0:indexToDelete], c.state.LRPs[indexToDelete+1:]...)
	c.workToCommit.Stops = append(c.workToCommit.Stops, stop)

	return nil
}

func (c *Cell) Commit() auctiontypes.Work {
	if len(c.workToCommit.Starts) == 0 && len(c.workToCommit.Stops) == 0 {
		return auctiontypes.Work{}
	}

	failedWork, err := c.client.Perform(c.workToCommit)
	if err != nil {
		return c.workToCommit
	}
	return failedWork
}

func (c *Cell) canHandleStartAuction(startAuction models.LRPStartAuction) error {
	if c.state.Stack != startAuction.DesiredLRP.Stack {
		return auctiontypes.ErrorStackMismatch
	}
	if c.state.AvailableResources.MemoryMB < startAuction.DesiredLRP.MemoryMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.DiskMB < startAuction.DesiredLRP.DiskMB {
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
