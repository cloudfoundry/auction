package algorithms

import (
	"math"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

/*

Get the bids from the subset of reps
    Pick the winner (lowest bid)
        Tell the winner to reserve
        	Compare the winner's new score to the previously received bids, if it exceeds a percentil threshold: repeat
*/

const percentile = 0.2

func CompareToPercentileAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int, []auctiontypes.AuctionEvent) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewStartAuctionInfoFromLRPStartAuction(auctionRequest.LRPStartAuction)
	events := []auctiontypes.AuctionEvent{}
	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		t := time.Now()
		//pick a subset
		repSubset := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPoolFraction, auctionRequest.Rules.MinBiddingPool)

		//get everyone's bid, if they're all full: bail
		numCommunications += len(repSubset)
		scores := client.BidForStartAuction(repSubset, auctionInfo)
		events = append(events, auctiontypes.AuctionEvent{"bid", time.Since(t), rounds, len(repSubset), ""})

		if scores.AllFailed() {
			events = append(events, auctiontypes.AuctionEvent{"all-full", 0, rounds, 0, ""})
			continue
		}

		t = time.Now()
		sortedScores := scores.FilterErrors().Shuffle().Sort()
		winner := sortedScores[0].Rep

		// tell the winner to reserve
		numCommunications += 1
		reservations := client.RebidThenTentativelyReserve([]string{winner}, auctionInfo)
		events = append(events, auctiontypes.AuctionEvent{"reserve", time.Since(t), rounds, 1, ""})

		if len(reservations) == 0 {
			events = append(events, auctiontypes.AuctionEvent{"reservation-failed", 0, rounds, 0, "empty"})
			continue
		}
		winnerRecast := reservations[0]
		//if the winner ran out of space: bail
		if winnerRecast.Error != "" {
			events = append(events, auctiontypes.AuctionEvent{"reservation-failed", 0, rounds, 0, winnerRecast.Error})
			continue
		}

		if len(sortedScores) > 1 {
			t = time.Now()
			index := int(math.Floor(float64(len(sortedScores)-2)*percentile)) + 1
			if sortedScores[index].Bid < winnerRecast.Bid {
				client.ReleaseReservation([]string{winner}, auctionInfo)
				events = append(events, auctiontypes.AuctionEvent{"release-reservation", 0, rounds, 0, ""})
				numCommunications += 1
				continue
			}
		}

		t = time.Now()
		numCommunications += 1
		client.Run(winner, auctionRequest.LRPStartAuction)
		events = append(events, auctiontypes.AuctionEvent{"run", time.Since(t), rounds, 1, ""})

		return winner, rounds, numCommunications, events
	}

	return "", rounds, numCommunications, events
}
