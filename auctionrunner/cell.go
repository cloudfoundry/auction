package auctionrunner

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var ErrorStackMismatch = errors.New("stack mismatch")
var ErrorInsufficientResources = errors.New("insuccifient resources")
var ErrorNothingToStop = errors.New("nothing to stop")

type Cell struct {
	client auctiontypes.AuctionRep
	state  auctiontypes.RepState

	workToCommit []auctiontypes.Work
}

func NewCell(client auctiontypes.AuctionRep, state auctiontypes.RepState) *Cell {
	return &Cell{
		client: client,
		state:  state,
	}
}

func (c *Cell) ScoreForStartAuction(startAuction models.LRPStartAuction) (float64, error) {
	if c.state.Stack != startAuction.DesiredLRP.Stack {
		return 0, ErrorStackMismatch
	}
	if c.state.AvailableResources.MemoryMB < startAuction.DesiredLRP.MemoryMB {
		return 0, ErrorInsufficientResources
	}
	if c.state.AvailableResources.DiskMB < startAuction.DesiredLRP.DiskMB {
		return 0, ErrorInsufficientResources
	}
	if c.state.AvailableResources.Containers < 1 {
		return 0, ErrorInsufficientResources
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
		return 0, nil, ErrorNothingToStop
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
	return nil
}

func (c *Cell) StopLRP(lrp models.StopLRPInstance) error {
	return nil
}

func (c *Cell) Commit() []auctiontypes.Work {
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
