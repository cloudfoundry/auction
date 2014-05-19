package auctioneer

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Pick an arbitrary rep
	Tell it to reserve
		If the reservation succeeds -- we have a winner

*/

func randomAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.AuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		randomPick := auctionRequest.RepGuids.RandomSubsetByCount(1)[0]
		result := client.ScoreThenTentativelyReserve([]string{randomPick}, auctionRequest.Instance)[0]
		numCommunications += 1
		if result.Error != "" {
			continue
		}

		client.Claim(randomPick, auctionRequest.Instance)
		numCommunications += 1

		return randomPick, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
