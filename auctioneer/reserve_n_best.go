package auctioneer

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Get the scores from the subset of reps
	Tell the top 5 to reserve
		Pick the best from that set and release the others

*/

func reserveNBestAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.AuctionRequest) (string, int, int) {
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

		// pick the top 5 winners
		winners := firstRoundScores.FilterErrors().Shuffle().Sort()
		max := 5
		if len(winners) < max {
			max = len(winners)
		}
		winners = winners[:max]

		//ask them to reserve
		numCommunications += len(winners)
		winners = client.ScoreThenTentativelyReserve(winners.Reps(), auctionRequest.Instance)
		//if they're all out of space, try again
		if winners.AllFailed() {
			continue
		}

		//order by score: the first is the winner, all others release
		orderedReps := winners.FilterErrors().Shuffle().Sort().Reps()

		numCommunications += len(winners)
		client.Claim(orderedReps[0], auctionRequest.Instance)
		if len(orderedReps) > 1 {
			client.ReleaseReservation(orderedReps[1:], auctionRequest.Instance)
		}

		return orderedReps[0], rounds, numCommunications
	}

	return "", rounds, numCommunications
}
