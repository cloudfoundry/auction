package auctioneer

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Get the scores from the subset of reps
    Pick the winner (lowest score)
        Tell the winner to reserve and the others to rescore
        	If the winner still has the lowest score we are done, otherwise, repeat

*/

func allRescoreAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.AuctionRequest) (string, int, int) {
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

		// tell the winner to reserve
		numCommunications += 1
		winnerRecast := client.ScoreThenTentativelyReserve([]string{winner.Rep}, auctionRequest.Instance)[0]

		//get everyone's score again
		secondRoundReps := firstRoundReps.Without(winner.Rep)
		numCommunications += len(secondRoundReps)
		secondRoundScores := client.Score(secondRoundReps, auctionRequest.Instance)

		//if the winner ran out of space: bail
		if winnerRecast.Error != "" {
			continue
		}

		// if the second place winner has a better score than the original winner: bail
		if !secondRoundScores.AllFailed() {
			secondPlace := secondRoundScores.FilterErrors().Shuffle().Sort()[0]
			if secondPlace.Score < winnerRecast.Score {
				client.ReleaseReservation([]string{winner.Rep}, auctionRequest.Instance)
				numCommunications += 1
				continue
			}
		}

		client.Claim(winner.Rep, auctionRequest.Instance)
		numCommunications += 1
		return winner.Rep, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
