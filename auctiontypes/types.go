package auctiontypes

import (
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type StartAuctionResult struct {
	LRPStartAuction models.LRPStartAuction
	Winner          string

	WaitTime time.Duration
}

type StopAuctionResult struct {
	LRPStopAuction models.LRPStopAuction
	Winner         string

	WaitTime time.Duration
}

type RepAddress struct {
	RepGuid string
	Address string
}

type AuctionRep interface {
	State() (RepState, error)
	Perform(Work) (Work, error)
}

//simulation-only interface
type SimulationAuctionRep interface {
	AuctionRep

	Reset() error
}

type Work struct {
	Starts []models.LRPStartAuction
	Stops  []models.StopLRPInstance
}

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

type Resources struct {
	DiskMB     int
	MemoryMB   int
	Containers int
}
