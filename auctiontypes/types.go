package auctiontypes

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/ifrit"
)

// Auction Runners

var ErrorStackMismatch = errors.New("stack mismatch")
var ErrorInsufficientResources = errors.New("insuccifient resources")
var ErrorNothingToStop = errors.New("nothing to stop")

type AuctionRunner interface {
	ifrit.Runner
	AddLRPStartAuction(models.LRPStartAuction)
	AddLRPStopAuction(models.LRPStopAuction)
}

type AuctionRunnerDelegate interface {
	FetchCellReps() (map[string]CellRep, error)
	DistributedBatch(AuctionResults)
}

type AuctionResults struct {
	SuccessfulStarts []StartAuction
	SuccessfulStops  []StopAuction
	FailedStarts     []StartAuction
	FailedStops      []StopAuction
}

// Start and Stop Auctions

type StartAuction struct {
	LRPStartAuction models.LRPStartAuction
	Winner          string
	Attempts        int

	QueueTime    time.Time
	WaitDuration time.Duration
}

func (s StartAuction) Identifier() string {
	return IdentifierForLRPStartAuction(s.LRPStartAuction)
}

func IdentifierForLRPStartAuction(start models.LRPStartAuction) string {
	return fmt.Sprintf("%s.%d.%s", start.DesiredLRP.ProcessGuid, start.Index, start.InstanceGuid)
}

type StopAuction struct {
	LRPStopAuction models.LRPStopAuction
	Winner         string
	Attempts       int

	QueueTime    time.Time
	WaitDuration time.Duration
}

func (s StopAuction) Identifier() string {
	return fmt.Sprintf("%s.%d", s.LRPStopAuction.ProcessGuid, s.LRPStopAuction.Index)
}

// Cell Representatives

type CellRep interface {
	State() (CellState, error)
	Perform(Work) (Work, error)
}

type SimulationCellRep interface {
	CellRep

	Reset() error
}

type Work struct {
	Starts []models.LRPStartAuction
	Stops  []models.ActualLRP
}

type CellState struct {
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
