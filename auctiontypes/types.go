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
var ErrorInsufficientResources = errors.New("insufficient resources")
var ErrorNothingToStop = errors.New("nothing to stop")

type AuctionRunner interface {
	ifrit.Runner
	AddLRPStartForAuction(models.LRPStart)
	AddTaskForAuction(models.Task)
}

type AuctionRunnerDelegate interface {
	FetchCellReps() (map[string]CellRep, error)
	DistributedBatch(AuctionResults)
}

type AuctionRequest struct {
	LRPStarts []LRPStartAuction
	Tasks     []TaskAuction
}

type AuctionResults struct {
	SuccessfulLRPStarts []LRPStartAuction
	SuccessfulTasks     []TaskAuction
	FailedLRPStarts     []LRPStartAuction
	FailedTasks         []TaskAuction
}

// LRPStart and Task Auctions

type AuctionRecord struct {
	Winner   string
	Attempts int

	QueueTime    time.Time
	WaitDuration time.Duration
}

type LRPStartAuction struct {
	LRPStart models.LRPStart
	AuctionRecord
}

func (s LRPStartAuction) Identifier() string {
	return IdentifierForLRPStartAuction(s.LRPStart)
}

func IdentifierForLRPStartAuction(start models.LRPStart) string {
	return IdentifierForLRP(start.DesiredLRP.ProcessGuid, start.Index)
}

func IdentifierForLRP(processGuid string, index int) string {
	return fmt.Sprintf("%s.%d", processGuid, index)
}

type TaskAuction struct {
	Task models.Task
	AuctionRecord
}

func (t TaskAuction) Identifier() string {
	return IdentifierForTask(t.Task)
}

func IdentifierForTask(t models.Task) string {
	return t.TaskGuid
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
	LRPStarts []models.LRPStart
	Tasks     []models.Task
}

type CellState struct {
	Stack              string
	AvailableResources Resources
	TotalResources     Resources
	LRPs               []LRP
	Tasks              []Task
}

type LRP struct {
	ProcessGuid string
	Index       int
	MemoryMB    int
	DiskMB      int
}

func (s LRP) Identifier() string {
	return IdentifierForLRP(s.ProcessGuid, s.Index)
}

type Task struct {
	TaskGuid string
	MemoryMB int
	DiskMB   int
}

type Resources struct {
	DiskMB     int
	MemoryMB   int
	Containers int
}
