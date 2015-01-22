package auctiontypes

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/diego_errors"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/ifrit"
)

// Auction Runners

var ErrorStackMismatch = errors.New(diego_errors.STACK_MISMATCH)
var ErrorInsufficientResources = errors.New(diego_errors.INSUFFICIENT_RESOURCES_MESSAGE)
var ErrorNothingToStop = errors.New("nothing to stop")

//go:generate counterfeiter -o fakes/fake_auction_runner.go . AuctionRunner
type AuctionRunner interface {
	ifrit.Runner
	ScheduleLRPsForAuctions([]models.LRPStartRequest)
	ScheduleTasksForAuctions([]models.Task)
}

type AuctionRunnerDelegate interface {
	FetchCellReps() (map[string]CellRep, error)
	AuctionCompleted(AuctionResults)
}

type AuctionRequest struct {
	LRPs  []LRPAuction
	Tasks []TaskAuction
}

type AuctionResults struct {
	SuccessfulLRPs  []LRPAuction
	SuccessfulTasks []TaskAuction
	FailedLRPs      []LRPAuction
	FailedTasks     []TaskAuction
}

// LRPStart and Task Auctions

type AuctionRecord struct {
	Winner   string
	Attempts int

	QueueTime    time.Time
	WaitDuration time.Duration

	PlacementError string
}

type LRPAuction struct {
	DesiredLRP models.DesiredLRP
	Index      int
	AuctionRecord
}

func (s LRPAuction) Identifier() string {
	return IdentifierForLRP(s.DesiredLRP.ProcessGuid, s.Index)
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
	LRPs  []LRPAuction
	Tasks []models.Task
}

type CellState struct {
	Stack              string
	AvailableResources Resources
	TotalResources     Resources
	LRPs               []LRP
	Tasks              []Task
	Zone               string
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
