package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Tell the subset of reps to reserve
    Pick the winner (lowest bid)
        Tell the winner to run and the others to release

*/
func allReserveAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewStartAuctionInfoFromLRPStartAuction(auctionRequest.LRPStartAuction)

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		firstRoundReps := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPoolFraction, auctionRequest.Rules.MinBiddingPool)

		//reserve everyone
		numCommunications += len(firstRoundReps)
		bids := client.RebidThenTentativelyReserve(firstRoundReps, auctionInfo)

		if bids.AllFailed() {
			continue
		}

		orderedReps := bids.FilterErrors().Shuffle().Sort().Reps()

		numCommunications += len(orderedReps)
		client.Run(orderedReps[0], auctionRequest.LRPStartAuction)
		if len(orderedReps) > 1 {
			client.ReleaseReservation(orderedReps[1:], auctionInfo)
		}

		return orderedReps[0], rounds, numCommunications
	}

	return "", rounds, numCommunications
}
