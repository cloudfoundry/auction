package algorithms

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

/*

Get the bids from the subset of reps
    Select the best

*/

func PickBestAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int, auctiontypes.AuctionEvents) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewStartAuctionInfoFromLRPStartAuction(auctionRequest.LRPStartAuction)
	events := auctiontypes.AuctionEvents{}

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		t := time.Now()
		//pick a subset
		repSubset := auctionRequest.RepAddresses.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPoolFraction, auctionRequest.Rules.MinBiddingPool)

		//get everyone's bid, if they're all full: bail
		numCommunications += len(repSubset)
		firstRoundScores := client.BidForStartAuction(repSubset, auctionInfo)
		events = append(events, auctiontypes.AuctionEvent{"bid", time.Since(t), rounds, len(repSubset), ""})

		if firstRoundScores.AllFailed() {
			events = append(events, auctiontypes.AuctionEvent{"all-full", 0, rounds, 0, ""})
			continue
		}

		t = time.Now()

		winner := firstRoundScores.FilterErrors().Shuffle().Sort()[0]
		numCommunications += 1
		reservations := client.RebidThenTentativelyReserve(auctionRequest.RepAddresses.Lookup(winner.Rep), auctionRequest.LRPStartAuction)
		events = append(events, auctiontypes.AuctionEvent{"reserve", time.Since(t), rounds, 1, ""})

		if len(reservations) == 0 {
			events = append(events, auctiontypes.AuctionEvent{"reservation-failed", 0, rounds, 0, "empty"})
			continue
		}
		if reservations[0].Error != "" {
			events = append(events, auctiontypes.AuctionEvent{"reservation-failed", 0, rounds, 0, reservations[0].Error})
			continue
		}

		t = time.Now()
		numCommunications += 1
		client.Run(auctionRequest.RepAddresses.AddressFor(winner.Rep), auctionRequest.LRPStartAuction)
		events = append(events, auctiontypes.AuctionEvent{"run", time.Since(t), rounds, 1, ""})

		return winner.Rep, rounds, numCommunications, events
	}

	return "", rounds, numCommunications, events
}
