package auctiontypes

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var InsufficientResources = errors.New("insufficient resources for instance")

type AuctionRunner interface {
	RunLRPStartAuction(auctionRequest AuctionRequest) (AuctionResult, error)
}

type AuctionRequest struct {
	LRPStartAuction models.LRPStartAuction `json:"a"`
	RepGuids        RepGuids               `json:"rg"`
	Rules           AuctionRules           `json:"r"`
}

type AuctionResult struct {
	LRPStartAuction   models.LRPStartAuction `json:"i"`
	Winner            string                 `json:"w"`
	NumRounds         int                    `json:"nr"`
	NumCommunications int                    `json:"nc"`
	BiddingDuration   time.Duration          `json:"bd"`
	Duration          time.Duration          `json:"d"`
}

type AuctionRules struct {
	Algorithm      string  `json:"alg"`
	MaxRounds      int     `json:"mr"`
	MaxBiddingPool float64 `json:"mb"`
}

type RepGuids []string

type ScoreResult struct {
	Rep   string  `json:"r"`
	Score float64 `json:"s"`
	Error string  `json:"e"`
}

type ScoreResults []ScoreResult

type Resources struct {
	DiskMB     int `json:"d"`
	MemoryMB   int `json:"m"`
	Containers int `json:"c,omitempty"`
}

type LRPAuctionInfo struct {
	AppGuid      string `json:"a"`
	InstanceGuid string `json:"i"`
	DiskMB       int    `json:"d"`
	MemoryMB     int    `json:"m"`
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
	ScoreThenTentativelyReserve(guids []string, instance LRPAuctionInfo) ScoreResults
	ReleaseReservation(guids []string, instance LRPAuctionInfo)
	Run(guid string, instance models.LRPStartAuction)
}

type TestRepPoolClient interface {
	RepPoolClient

	TotalResources(guid string) Resources
	LRPAuctionInfos(guid string) []LRPAuctionInfo
	SetLRPAuctionInfos(guid string, instances []LRPAuctionInfo)
	Reset(guid string)
}
