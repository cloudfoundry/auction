package algorithms

import (
	"math"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

/*

Get the bids from the subset of reps
    Pick the winner (lowest bid)
        Tell the winner to reserve
        	Compare the winner's new score to the previously received bids, if it exceeds a percentil threshold: repeat
*/

const percentile = 0.2

func CompareToPercentileAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewStartAuctionInfoFromLRPStartAuction(auctionRequest.LRPStartAuction)
	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		repSubset := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPoolFraction, auctionRequest.Rules.MinBiddingPool)

		//get everyone's bid, if they're all full: bail
		numCommunications += len(repSubset)
		scores := client.BidForStartAuction(repSubset, auctionInfo)
		if scores.AllFailed() {
			continue
		}

		sortedScores := scores.FilterErrors().Shuffle().Sort()
		winner := sortedScores[0].Rep

		// tell the winner to reserve
		numCommunications += 1
		reservations := client.RebidThenTentativelyReserve([]string{winner}, auctionInfo)
		if len(reservations) == 0 {
			continue
		}
		winnerRecast := reservations[0]
		//if the winner ran out of space: bail
		if winnerRecast.Error != "" {
			continue
		}

		if len(sortedScores) > 1 {
			index := int(math.Floor(float64(len(sortedScores)-2)*percentile)) + 1
			if sortedScores[index].Bid < winnerRecast.Bid {
				client.ReleaseReservation([]string{winner}, auctionInfo)
				numCommunications += 1
				continue
			}
		}

		client.Run(winner, auctionRequest.LRPStartAuction)
		numCommunications += 1
		return winner, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
