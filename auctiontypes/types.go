package auctiontypes

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/auctioneer"
	"github.com/cloudfoundry-incubator/rep"
	"github.com/tedsuo/ifrit"
)

// Auction Runners

var ErrorCellMismatch = errors.New("found no compatible cell")
var ErrorNothingToStop = errors.New("nothing to stop")
var ErrorCellCommunication = errors.New("unable to communicate to compatible cells")

//go:generate counterfeiter -o fakes/fake_auction_runner.go . AuctionRunner
type AuctionRunner interface {
	ifrit.Runner
	ScheduleLRPsForAuctions([]auctioneer.LRPStartRequest)
	ScheduleTasksForAuctions([]auctioneer.TaskStartRequest)
}

type AuctionRunnerDelegate interface {
	FetchCellReps() (map[string]rep.Client, error)
	AuctionCompleted(AuctionResults)
}

//go:generate counterfeiter -o fakes/fake_metric_emitter.go . AuctionMetricEmitterDelegate
type AuctionMetricEmitterDelegate interface {
	FetchStatesCompleted(time.Duration)
	FailedCellStateRequest()
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

func NewAuctionRecord(now time.Time) AuctionRecord {
	return AuctionRecord{QueueTime: now}
}

type LRPAuction struct {
	rep.LRP
	AuctionRecord
}

func NewLRPAuction(lrp rep.LRP, now time.Time) LRPAuction {
	return LRPAuction{
		lrp,
		NewAuctionRecord(now),
	}
}

func (a *LRPAuction) Copy() LRPAuction {
	return LRPAuction{a.LRP.Copy(), a.AuctionRecord}
}

type TaskAuction struct {
	rep.Task
	AuctionRecord
}

func NewTaskAuction(task rep.Task, now time.Time) TaskAuction {
	return TaskAuction{
		task,
		NewAuctionRecord(now),
	}
}

func (a *TaskAuction) Copy() TaskAuction {
	return TaskAuction{a.Task.Copy(), a.AuctionRecord}
}
