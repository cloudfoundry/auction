package auctionrunner

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrunner/algorithms"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

var AllBiddersFull = errors.New("all the bidders were full")

var DefaultStartAuctionRules = auctiontypes.StartAuctionRules{
	Algorithm:              "compare_to_percentile",
	MaxRounds:              100,
	MaxBiddingPoolFraction: 0.2,
	MinBiddingPool:         10,
	ComparisonPercentile:   0.2,
}

type auctionRunner struct {
	client auctiontypes.RepPoolClient
}

func New(client auctiontypes.RepPoolClient) *auctionRunner {
	return &auctionRunner{
		client: client,
	}
}

func (a *auctionRunner) RunLRPStartAuction(auctionRequest auctiontypes.StartAuctionRequest) (auctiontypes.StartAuctionResult, error) {
	result := auctiontypes.StartAuctionResult{
		LRPStartAuction:  auctionRequest.LRPStartAuction,
		AuctionStartTime: time.Now(),
	}

	t := time.Now()
	switch auctionRequest.Rules.Algorithm {
	case "all_rebid":
		result.Winner, result.NumRounds, result.NumCommunications, result.Events = algorithms.AllRebidAuction(a.client, auctionRequest)
	case "compare_to_percentile":
		result.Winner, result.NumRounds, result.NumCommunications, result.Events = algorithms.CompareToPercentileAuction(a.client, auctionRequest)
	case "pick_best":
		result.Winner, result.NumRounds, result.NumCommunications, result.Events = algorithms.PickBestAuction(a.client, auctionRequest)
	case "random":
		result.Winner, result.NumRounds, result.NumCommunications, result.Events = algorithms.RandomAuction(a.client, auctionRequest)
	default:
		panic("unkown algorithm " + auctionRequest.Rules.Algorithm)
	}
	result.BiddingDuration = time.Since(t)

	if result.Winner == "" {
		return result, auctiontypes.InsufficientResources
	}

	return result, nil
}

func (a *auctionRunner) RunLRPStopAuction(auctionRequest auctiontypes.StopAuctionRequest) (auctiontypes.StopAuctionResult, error) {
	result := auctiontypes.StopAuctionResult{
		LRPStopAuction: auctionRequest.LRPStopAuction,
	}

	var err error
	t := time.Now()
	result.Winner, result.NumCommunications, err = algorithms.StopAuction(a.client, auctionRequest)
	result.BiddingDuration = time.Since(t)

	return result, err
}
