package auctiontypes

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

//errors
var InsufficientResources = errors.New("insufficient resources for instance")
var NothingToStop = errors.New("found nothing to stop")

//AuctionRunner
type AuctionRunner interface {
	RunLRPStartAuction(auctionRequest StartAuctionRequest) (StartAuctionResult, error)
	RunLRPStopAuction(auctionRequest StopAuctionRequest) (StopAuctionResult, error)
}

type StartAuctionRequest struct {
	LRPStartAuction models.LRPStartAuction
	RepGuids        RepGuids
	Rules           StartAuctionRules
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

type StartAuctionRules struct {
	Algorithm      string
	MaxRounds      int
	MaxBiddingPool float64
}

type RepGuids []string

type RepPoolClient interface {
	BidForStartAuction(guids []string, startAuctionInfo StartAuctionInfo) StartAuctionBids
	BidForStopAuction(guids []string, stopAuctionInfo StopAuctionInfo) StopAuctionBids
	RebidThenTentativelyReserve(guids []string, startAuctionInfo StartAuctionInfo) StartAuctionBids
	ReleaseReservation(guids []string, startAuctionInfo StartAuctionInfo)
	Run(guid string, startAuctionInfo models.LRPStartAuction)
	Stop(guid string, instanceGuid string)
}

type AuctionRepDelegate interface {
	RemainingResources() (Resources, error)
	TotalResources() (Resources, error)
	NumInstancesForProcessGuid(guid string) (int, error)
	InstanceGuidsForProcessGuidAndIndex(guid string, index int) ([]string, error)

	Reserve(startAuctionInfo StartAuctionInfo) error
	ReleaseReservation(startAuctionInfo StartAuctionInfo) error
	Run(startAuction models.LRPStartAuction) error
	Stop(instanceGuid string) error
}

//simulation-only interface
type SimulationRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) Resources
	SimulatedInstances(guid string) []SimulatedInstance
	SetSimulatedInstances(guid string, instances []SimulatedInstance)
	Reset(guid string)
}

//simulation-only interface
type SimulationAuctionRepDelegate interface {
	AuctionRepDelegate
	SetSimulatedInstances(instances []SimulatedInstance)
	SimulatedInstances() []SimulatedInstance
}

func NewStartAuctionInfoFromLRPStartAuction(auction models.LRPStartAuction) StartAuctionInfo {
	return StartAuctionInfo{
		ProcessGuid:  auction.ProcessGuid,
		InstanceGuid: auction.InstanceGuid,
		DiskMB:       auction.DiskMB,
		MemoryMB:     auction.MemoryMB,
	}
}

func NewStopAuctionInfoFromLRPStopAuction(auction models.LRPStopAuction) StopAuctionInfo {
	return StopAuctionInfo{
		ProcessGuid: auction.ProcessGuid,
		Index:       auction.Index,
	}
}

type StartAuctionBid struct {
	Rep   string
	Bid   float64
	Error string
}

type StartAuctionBids []StartAuctionBid

type StopAuctionBid struct {
	Rep           string
	InstanceGuids []string
	Bid           float64
	Error         string
}

type StopAuctionBids []StopAuctionBid

type Resources struct {
	DiskMB     int
	MemoryMB   int
	Containers int
}

type StartAuctionInfo struct {
	ProcessGuid  string
	InstanceGuid string
	DiskMB       int
	MemoryMB     int
}

type StopAuctionInfo struct {
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
