package auctioneer

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

var AllBiddersFull = errors.New("all the bidders were full")

var DefaultRules = auctiontypes.AuctionRules{
	Algorithm:      "reserve_n_best",
	MaxRounds:      100,
	MaxBiddingPool: 0.2,
}

type auctionRunner struct {
	client auctiontypes.RepPoolClient
}

func New(client auctiontypes.RepPoolClient) *auctionRunner {
	return &auctionRunner{
		client: client,
	}
}

func (a *auctionRunner) RunLRPStartAuction(auctionRequest auctiontypes.AuctionRequest) (auctiontypes.AuctionResult, error) {
	result := auctiontypes.AuctionResult{
		Instance: auctionRequest.Instance,
	}

	t := time.Now()
	switch auctionRequest.Rules.Algorithm {
	case "all_rescore":
		result.Winner, result.NumRounds, result.NumCommunications = allRescoreAuction(a.client, auctionRequest)
	case "all_reserve":
		result.Winner, result.NumRounds, result.NumCommunications = allReserveAuction(a.client, auctionRequest)
	case "pick_among_best":
		result.Winner, result.NumRounds, result.NumCommunications = pickAmongBestAuction(a.client, auctionRequest)
	case "pick_best":
		result.Winner, result.NumRounds, result.NumCommunications = pickBestAuction(a.client, auctionRequest)
	case "reserve_n_best":
		result.Winner, result.NumRounds, result.NumCommunications = reserveNBestAuction(a.client, auctionRequest)
	case "random":
		result.Winner, result.NumRounds, result.NumCommunications = randomAuction(a.client, auctionRequest)
	default:
		panic("unkown algorithm " + auctionRequest.Rules.Algorithm)
	}
	result.BiddingDuration = time.Since(t)

	return result, nil
}
