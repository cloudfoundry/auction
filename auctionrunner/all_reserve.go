package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Tell the subset of reps to reserve
    Pick the winner (lowest score)
        Tell the winner to run and the others to release

*/
func allReserveAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewLRPAuctionInfo(auctionRequest.LRPStartAuction)

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		firstRoundReps := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPool)

		//reserve everyone
		numCommunications += len(firstRoundReps)
		scores := client.ScoreThenTentativelyReserve(firstRoundReps, auctionInfo)

		if scores.AllFailed() {
			continue
		}

		orderedReps := scores.FilterErrors().Shuffle().Sort().Reps()

		numCommunications += len(orderedReps)
		client.Run(orderedReps[0], auctionRequest.LRPStartAuction)
		if len(orderedReps) > 1 {
			client.ReleaseReservation(orderedReps[1:], auctionInfo)
		}

		return orderedReps[0], rounds, numCommunications
	}

	return "", rounds, numCommunications
}
