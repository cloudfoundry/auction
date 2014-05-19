package auctioneer

import "github.com/onsi/auction/types"

/*

Get the scores from the subset of reps
    Select the best

*/

func pickBestAuction(client types.RepPoolClient, auctionRequest types.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		//pick a subset
		firstRoundReps := auctionRequest.RepGuids.RandomSubsetByFraction(auctionRequest.Rules.MaxBiddingPool)

		//get everyone's score, if they're all full: bail
		numCommunications += len(firstRoundReps)
		firstRoundScores := client.Score(firstRoundReps, auctionRequest.Instance)
		if firstRoundScores.AllFailed() {
			continue
		}

		winner := firstRoundScores.FilterErrors().Shuffle().Sort()[0]

		result := client.ScoreThenTentativelyReserve([]string{winner.Rep}, auctionRequest.Instance)[0]
		numCommunications += 1
		if result.Error != "" {
			continue
		}

		client.Claim(winner.Rep, auctionRequest.Instance)
		numCommunications += 1

		return winner.Rep, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
