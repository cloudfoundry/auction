package auctionrunner

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var StackMismatch = errors.New("stack mismatch")
var InsufficientResources = errors.New("insuccifient resources")

type Cell struct {
	client auctiontypes.AuctionRep
	state  auctiontypes.RepState

	workToCommit []auctiontypes.Work
}

/*
type RepState struct {
    Stack              string
    AvailableResources Resources
    TotalResources     Resources
    LRPs               []LRP
}

type LRP struct {
    ProcessGuid  string
    InstanceGuid string
    Index        int
    MemoryMB     int
    DiskMB       int
}
*/

func NewCell(client auctiontypes.AuctionRep, state auctiontypes.RepState) *Cell {
	return &Cell{
		client: client,
		state:  state,
	}
}

func (c *Cell) ScoreToStartLRP(startAuction models.LRPStartAuction) (float64, error) {
	if c.state.Stack != startAuction.DesiredLRP.Stack {
		return 0, StackMismatch
	}
	if c.state.AvailableResources.MemoryMB < startAuction.DesiredLRP.MemoryMB {
		return 0, InsufficientResources
	}
	if c.state.AvailableResources.DiskMB < startAuction.DesiredLRP.DiskMB {
		return 0, InsufficientResources
	}
	if c.state.AvailableResources.Containers < 1 {
		return 0, InsufficientResources
	}

	remainingMemory := c.state.AvailableResources.MemoryMB - startAuction.DesiredLRP.MemoryMB
	remainingDisk := c.state.AvailableResources.DiskMB - startAuction.DesiredLRP.DiskMB
	remainingContainers := c.state.AvailableResources.Containers - 1

	fractionUsedMemory := 1.0 - float64(remainingMemory)/float64(c.state.TotalResources.MemoryMB)
	fractionUsedDisk := 1.0 - float64(remainingDisk)/float64(c.state.TotalResources.DiskMB)
	fractionUsedContainers := 1.0 - float64(remainingContainers)/float64(c.state.TotalResources.Containers)

	resourceScore := (fractionUsedMemory + fractionUsedDisk + fractionUsedContainers) / 3.0

	numberOfInstancesWithMatchingProcessGuid := float64(0)
	for _, lrp := range c.state.LRPs {
		if lrp.ProcessGuid == startAuction.DesiredLRP.ProcessGuid {
			numberOfInstancesWithMatchingProcessGuid++
		}
	}

	resourceScore += numberOfInstancesWithMatchingProcessGuid

	return resourceScore, nil
}

func (c *Cell) ScoreToStopLRP(stopAuction models.LRPStopAuction) (float64, []string, error) {
	return 0, nil, nil
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
