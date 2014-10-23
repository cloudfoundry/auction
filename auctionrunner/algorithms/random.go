package algorithms

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

/*

Pick an arbitrary rep
	Tell it to reserve
		If the reservation succeeds -- we have a winner

*/

func RandomAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StartAuctionRequest) (string, int, int, auctiontypes.AuctionEvents) {
	rounds, numCommunications := 1, 0
	events := auctiontypes.AuctionEvents{}

	for ; rounds <= auctionRequest.Rules.MaxRounds; rounds++ {
		t := time.Now()
		randomPick := auctionRequest.RepGuids.RandomSubsetByCount(1)[0]

		numCommunications += 1
		reservations := client.RebidThenTentativelyReserve([]string{randomPick}, auctionRequest.LRPStartAuction)
		events = append(events, auctiontypes.AuctionEvent{"reserve", time.Since(t), rounds, 1, ""})

		if len(reservations) == 0 {
			events = append(events, auctiontypes.AuctionEvent{"reservation-failed", 0, rounds, 0, "empty"})
			continue
		}
		if reservations[0].Error != "" {
			events = append(events, auctiontypes.AuctionEvent{"reservation-failed", 0, rounds, 0, reservations[0].Error})
			continue
		}

		t = time.Now()
		numCommunications += 1
		client.Run(randomPick, auctionRequest.LRPStartAuction)
		events = append(events, auctiontypes.AuctionEvent{"run", time.Since(t), rounds, 1, ""})

		return randomPick, rounds, numCommunications, events
	}

	return "", rounds, numCommunications, events
}
