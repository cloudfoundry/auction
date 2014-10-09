package auctiontypes

import (
	"errors"
	"fmt"
	"strings"
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
	RepAddresses    RepAddresses
	Rules           StartAuctionRules
}

type StartAuctionResult struct {
	LRPStartAuction   models.LRPStartAuction
	Winner            string
	NumRounds         int
	NumCommunications int

	AuctionStartTime time.Time
	BiddingDuration  time.Duration
	Duration         time.Duration
	Events           AuctionEvents
}

type AuctionEvents []AuctionEvent

func (a AuctionEvents) String() string {
	s := ""
	round := 0
	for _, event := range a {
		if round != event.Round {
			s += fmt.Sprintf("%d:\n", event.Round)
			round = event.Round
		}
		components := []string{event.Event}
		if event.Duration > 0 {
			components = append(components, event.Duration.String())
		}
		if event.Communication > 0 {
			components = append(components, fmt.Sprintf("+%d", event.Communication))
		}
		if event.Info != "" {
			components = append(components, event.Info)
		}

		s += "  " + strings.Join(components, " ") + "\n"
	}

	return s
}

type AuctionEvent struct {
	Event         string
	Duration      time.Duration
	Round         int
	Communication int
	Info          string
}

type StopAuctionRequest struct {
	LRPStopAuction models.LRPStopAuction
	RepAddresses   RepAddresses
}

type StopAuctionResult struct {
	LRPStopAuction    models.LRPStopAuction
	Winner            string
	NumCommunications int
	BiddingDuration   time.Duration
	Duration          time.Duration
}

type StartAuctionRules struct {
	Algorithm              string
	MaxRounds              int
	MaxBiddingPoolFraction float64
	MinBiddingPool         int
	ComparisonPercentile   float64
}

type RepAddress struct {
	RepGuid string
	Address string
}

type RepAddresses []RepAddress

type RepPoolClient interface {
	BidForStartAuction(repAddresses []RepAddress, startAuctionInfo StartAuctionInfo) StartAuctionBids
	BidForStopAuction(repAddresses []RepAddress, stopAuctionInfo StopAuctionInfo) StopAuctionBids
	RebidThenTentativelyReserve(repAddresses []RepAddress, startAuctionInfo models.LRPStartAuction) StartAuctionBids
	ReleaseReservation(repAddresses []RepAddress, startAuctionInfo models.LRPStartAuction)
	Run(repAddress RepAddress, startAuctionInfo models.LRPStartAuction)
	Stop(repAddress RepAddress, stopInstance models.StopLRPInstance)
}

type AuctionRep interface {
	Guid() string
	BidForStartAuction(startAuctionInfo StartAuctionInfo) (float64, error)
	BidForStopAuction(stopAuctionInfo StopAuctionInfo) (float64, []string, error)
	RebidThenTentativelyReserve(startAuctionInfo models.LRPStartAuction) (float64, error)
	ReleaseReservation(startAuctionInfo models.LRPStartAuction) error

	Run(startAuction models.LRPStartAuction) error
	Stop(stopInstance models.StopLRPInstance) error
}

type AuctionRepDelegate interface {
	RemainingResources() (Resources, error)
	TotalResources() (Resources, error)
	NumInstancesForProcessGuid(processGuid string) (int, error)
	InstanceGuidsForProcessGuidAndIndex(processGuid string, index int) ([]string, error)

	Reserve(startAuction models.LRPStartAuction) error
	ReleaseReservation(startAuction models.LRPStartAuction) error
	Run(startAuction models.LRPStartAuction) error
	Stop(stopInstance models.StopLRPInstance) error
}

//simulation-only interface
type SimulationRepPoolClient interface {
	RepPoolClient

	TotalResources(repAddress RepAddress) Resources
	SimulatedInstances(repAddress RepAddress) []SimulatedInstance
	SetSimulatedInstances(repAddress RepAddress, instances []SimulatedInstance)
	Reset(repAddress RepAddress)
}

//simulation-only interface
type SimulationAuctionRep interface {
	AuctionRep

	Reset()
	SetSimulatedInstances(instances []SimulatedInstance)
	SimulatedInstances() []SimulatedInstance
	TotalResources() Resources
}

//simulation-only interface
type SimulationAuctionRepDelegate interface {
	AuctionRepDelegate
	SetSimulatedInstances(instances []SimulatedInstance)
	SimulatedInstances() []SimulatedInstance
}

func NewStartAuctionInfoFromLRPStartAuction(auction models.LRPStartAuction) StartAuctionInfo {
	return StartAuctionInfo{
		ProcessGuid: auction.DesiredLRP.ProcessGuid,
		DiskMB:      auction.DesiredLRP.DiskMB,
		MemoryMB:    auction.DesiredLRP.MemoryMB,

		InstanceGuid: auction.InstanceGuid,
		Index:        auction.Index,
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
	Index        int
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
