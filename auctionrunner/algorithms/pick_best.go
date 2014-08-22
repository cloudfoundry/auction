package algorithms

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Get the bids from the subset of reps
    Select the best

*/

func PickBestAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int) {
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

		result := client.RebidThenTentativelyReserve([]string{winner.Rep}, auctionInfo)[0]
		numCommunications += 1
		if result.Error != "" {
			continue
		}

		client.Run(winner.Rep, auctionRequest.LRPStartAuction)
		numCommunications += 1

		return winner.Rep, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
