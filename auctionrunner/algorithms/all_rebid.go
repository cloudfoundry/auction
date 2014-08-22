package algorithms

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Get the bids from the subset of reps
    Pick the winner (lowest bid)
        Tell the winner to reserve and the others to rebid
        	If the winner still has the lowest bid we are done, otherwise, repeat
*/

func AllRebidAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int) {
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
			//rescore the runner ups
			repsToRescore := sortedScores[1:len(sortedScores)].Reps()
			numCommunications += len(repsToRescore)
			secondRoundScores := client.BidForStartAuction(repsToRescore, auctionInfo)

			// if the second place winner has a better bid than the original winner: bail
			if !secondRoundScores.AllFailed() {
				secondPlace := secondRoundScores.FilterErrors().Sort()[0]
				if secondPlace.Bid < winnerRecast.Bid && rounds < auctionRequest.Rules.MaxRounds {
					client.ReleaseReservation([]string{winner}, auctionInfo)
					numCommunications += 1
					continue
				}
			}
		}

		client.Run(winner, auctionRequest.LRPStartAuction)
		numCommunications += 1
		return winner, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
