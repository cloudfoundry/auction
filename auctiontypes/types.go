package auctiontypes

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var InsufficientResources = errors.New("insufficient resources for instance")

type AuctionRunner interface {
	RunLRPStartAuction(auctionRequest StartAuctionRequest) (StartAuctionResult, error)
	RunLRPStopAuction(auctionRequest StopAuctionRequest)
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

type LRPAuctionInfo struct {
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

func NewLRPAuctionInfo(info models.LRPStartAuction) LRPAuctionInfo {
	return LRPAuctionInfo{
		AppGuid:      info.ProcessGuid,
		InstanceGuid: info.InstanceGuid,
		DiskMB:       info.DiskMB,
		MemoryMB:     info.MemoryMB,
	}
}

type RepPoolClient interface {
	Score(guids []string, instance LRPAuctionInfo) ScoreResults
	StopScore(guids []string, stopAuctionInfo LRPStopAuctionInfo) StopScoreResults
	ScoreThenTentativelyReserve(guids []string, instance LRPAuctionInfo) ScoreResults
	ReleaseReservation(guids []string, instance LRPAuctionInfo)
	Run(guid string, instance models.LRPStartAuction)
	Stop(guid string, instanceGuid string)
}

type TestRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) Resources
	SimulatedInstances(guid string) []SimulatedInstance
	SetSimulatedInstances(guid string, instances []SimulatedInstance)
	Reset(guid string)
}
