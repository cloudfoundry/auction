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

type AuctionRequest struct {
	LRPStarts []LRPStartAuction
	LRPStops  []LRPStopAuction
	Tasks     []TaskAuction
}

type AuctionResults struct {
	SuccessfulLRPStarts []LRPStartAuction
	SuccessfulLRPStops  []LRPStopAuction
	SuccessfulTasks     []TaskAuction
	FailedLRPStarts     []LRPStartAuction
	FailedLRPStops      []LRPStopAuction
	FailedTasks         []TaskAuction
}

// Start, Stop, and Task Auctions

type LRPStartAuction struct {
	LRPStartAuction models.LRPStartAuction
	Winner          string
	Attempts        int

	QueueTime    time.Time
	WaitDuration time.Duration
}

func (s LRPStartAuction) Identifier() string {
	return IdentifierForLRPStartAuction(s.LRPStartAuction)
}

func IdentifierForLRPStartAuction(start models.LRPStartAuction) string {
	return fmt.Sprintf("%s.%d.%s", start.DesiredLRP.ProcessGuid, start.Index, start.InstanceGuid)
}

type LRPStopAuction struct {
	LRPStopAuction models.LRPStopAuction
	Winner         string
	Attempts       int

	QueueTime    time.Time
	WaitDuration time.Duration
}

func (s LRPStopAuction) Identifier() string {
	return fmt.Sprintf("%s.%d", s.LRPStopAuction.ProcessGuid, s.LRPStopAuction.Index)
}

type TaskAuction struct {
	Task     models.Task
	Winner   string
	Attempts int

	QueueTime    time.Time
	WaitDuration time.Duration
}

func (t TaskAuction) Identifier() string {
	return t.Task.TaskGuid
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
	LRPStarts []models.LRPStartAuction
	LRPStops  []models.ActualLRP
	Tasks     []models.Task
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
