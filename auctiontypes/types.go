package auctiontypes

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var InsufficientResources = errors.New("insufficient resources for instance")
var NothingToStop = errors.New("found nothing to stop")

type AuctionRunner interface {
	RunLRPStartAuction(auctionRequest StartAuctionRequest) (StartAuctionResult, error)
	RunLRPStopAuction(auctionRequest StopAuctionRequest) (StopAuctionResult, error)
}

type RepPoolClient interface {
	Score(guids []string, startAuctionInfo LRPStartAuctionInfo) ScoreResults
	StopScore(guids []string, stopAuctionInfo LRPStopAuctionInfo) StopScoreResults
	ScoreThenTentativelyReserve(guids []string, startAuctionInfo LRPStartAuctionInfo) ScoreResults
	ReleaseReservation(guids []string, startAuctionInfo LRPStartAuctionInfo)
	Run(guid string, startAuctionInfo models.LRPStartAuction)
	Stop(guid string, instanceGuid string)
}

type AuctionRepDelegate interface {
	RemainingResources() (Resources, error)
	TotalResources() (Resources, error)
	NumInstancesForAppGuid(guid string) (int, error)
	InstanceGuidsForProcessGuidAndIndex(guid string, index int) ([]string, error)

	Reserve(startAuctionInfo LRPStartAuctionInfo) error
	ReleaseReservation(startAuctionInfo LRPStartAuctionInfo) error
	Run(startAuction models.LRPStartAuction) error
	Stop(instanceGuid string) error
}

//Used in simulation
type SimulationRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) Resources
	SimulatedInstances(guid string) []SimulatedInstance
	SetSimulatedInstances(guid string, instances []SimulatedInstance)
	Reset(guid string)
}

//Used in simulation
type SimulationAuctionRepDelegate interface {
	AuctionRepDelegate
	SetSimulatedInstances(instances []SimulatedInstance)
	SimulatedInstances() []SimulatedInstance
}

type StartAuctionRequest struct {
	LRPStartAuction models.LRPStartAuction
	RepGuids        RepGuids
	Rules           AuctionRules
}

type StartAuctionResult struct {
	LRPStartAuction   models.LRPStartAuction
	Winner            string
	NumRounds         int
	NumCommunications int
	BiddingDuration   time.Duration
	Duration          time.Duration
}

type StopAuctionRequest struct {
	LRPStopAuction models.LRPStopAuction
	RepGuids       RepGuids
}

type StopAuctionResult struct {
	LRPStopAuction    models.LRPStopAuction
	Winner            string
	NumCommunications int
	BiddingDuration   time.Duration
	Duration          time.Duration
}

type AuctionRules struct {
	Algorithm      string
	MaxRounds      int
	MaxBiddingPool float64
}

type RepGuids []string

type ScoreResult struct {
	Rep   string
	Score float64
	Error string
}

type ScoreResults []ScoreResult

type StopScoreResult struct {
	Rep           string
	InstanceGuids []string
	Score         float64
	Error         string
}

type StopScoreResults []StopScoreResult

type Resources struct {
	DiskMB     int
	MemoryMB   int
	Containers int
}

type LRPStartAuctionInfo struct {
	AppGuid      string
	InstanceGuid string
	DiskMB       int
	MemoryMB     int
}

type LRPStopAuctionInfo struct {
	ProcessGuid string
	Index       int
}

type SimulatedInstance struct {
	ProcessGuid  string
	InstanceGuid string
	Index        int
	MemoryMB     int
	DiskMB       int
}

func NewLRPStartAuctionInfo(info models.LRPStartAuction) LRPStartAuctionInfo {
	return LRPStartAuctionInfo{
		AppGuid:      info.ProcessGuid,
		InstanceGuid: info.InstanceGuid,
		DiskMB:       info.DiskMB,
		MemoryMB:     info.MemoryMB,
	}
}
