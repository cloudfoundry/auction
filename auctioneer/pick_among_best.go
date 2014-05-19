package auctioneer

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Get the scores from the subset of reps
	Select the best 5
		Pick a winner randomly from that set

*/

func pickAmongBestAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewLRPAuctionInfo(auctionRequest.LRPStartAuction)

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		firstRoundReps := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPool)

		//get everyone's score, if they're all full: bail
		numCommunications += len(firstRoundReps)
		firstRoundScores := client.Score(firstRoundReps, auctionInfo)
		if firstRoundScores.AllFailed() {
			continue
		}

		top5Winners := firstRoundScores.FilterErrors().Shuffle().Sort()[:5]

		winner := top5Winners.Shuffle()[0]

		result := client.ScoreThenTentativelyReserve([]string{winner.Rep}, auctionInfo)[0]
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
