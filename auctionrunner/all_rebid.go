package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Get the bids from the subset of reps
    Pick the winner (lowest bid)
        Tell the winner to reserve and the others to rebid
        	If the winner still has the lowest bid we are done, otherwise, repeat

*/

func allRebidAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewStartAuctionInfoFromLRPStartAuction(auctionRequest.LRPStartAuction)

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		firstRoundReps := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPoolFraction, auctionRequest.Rules.MinBiddingPool)

		//get everyone's bid, if they're all full: bail
		numCommunications += len(firstRoundReps)
		firstRoundScores := client.BidForStartAuction(firstRoundReps, auctionInfo)
		if firstRoundScores.AllFailed() {
			continue
		}

		winner := firstRoundScores.FilterErrors().Shuffle().Sort()[0]

		// tell the winner to reserve
		numCommunications += 1
		winnerRecast := client.RebidThenTentativelyReserve([]string{winner.Rep}, auctionInfo)[0]

		//get everyone's bid again
		secondRoundReps := firstRoundReps.Without(winner.Rep)
		numCommunications += len(secondRoundReps)
		secondRoundScores := client.BidForStartAuction(secondRoundReps, auctionInfo)

		//if the winner ran out of space: bail
		if winnerRecast.Error != "" {
			continue
		}

		// if the second place winner has a better bid than the original winner: bail
		if !secondRoundScores.AllFailed() {
			secondPlace := secondRoundScores.FilterErrors().Shuffle().Sort()[0]
			if secondPlace.Bid < winnerRecast.Bid {
				client.ReleaseReservation([]string{winner.Rep}, auctionInfo)
				numCommunications += 1
				continue
			}
		}

		client.Run(winner.Rep, auctionRequest.LRPStartAuction)
		numCommunications += 1
		return winner.Rep, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
