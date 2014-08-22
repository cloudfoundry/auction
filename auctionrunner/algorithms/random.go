package algorithms

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

/*

Pick an arbitrary rep
	Tell it to reserve
		If the reservation succeeds -- we have a winner

*/

func RandomAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int) {
	rounds, numCommunications := 1, 0
	auctionInfo := auctiontypes.NewStartAuctionInfoFromLRPStartAuction(auctionRequest.LRPStartAuction)

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		randomPick := auctionRequest.RepGuids.RandomSubsetByCount(1)[0]
		result := client.RebidThenTentativelyReserve([]string{randomPick}, auctionInfo)[0]
		numCommunications += 1
		if result.Error != "" {
			continue
		}

		client.Run(randomPick, auctionRequest.LRPStartAuction)
		numCommunications += 1

		return randomPick, rounds, numCommunications
	}

	return "", rounds, numCommunications
}
