package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

func stopAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StopAuctionRequest) (string, int, error) {
	numCommunication := 0

	stopAuctionInfo := auctiontypes.StopAuctionInfo{
		ProcessGuid: auctionRequest.LRPStopAuction.ProcessGuid,
		Index:       auctionRequest.LRPStopAuction.Index,
	}

	numCommunication += len(auctionRequest.RepGuids)
	stopAuctionBids := client.BidForStopAuction(auctionRequest.RepGuids, stopAuctionInfo)
	stopAuctionBids = stopAuctionBids.FilterErrors()

	instanceGuids := stopAuctionBids.InstanceGuids()
	if len(instanceGuids) <= 1 {
		return "", numCommunication, auctiontypes.NothingToStop
	}

	stopAuctionBids = stopAuctionBids.Shuffle()

	var repGuidWithLoneRemainingInstance string
	lowestScore := 1e9

	for _, stopAuctionBid := range stopAuctionBids {
		bidIfRepGuidWins := stopAuctionBid.Bid - float64(len(stopAuctionBid.InstanceGuids)) + 1
		if bidIfRepGuidWins < lowestScore {
			lowestScore = bidIfRepGuidWins
			repGuidWithLoneRemainingInstance = stopAuctionBid.Rep
		}
	}

	wg := &sync.WaitGroup{}
	for _, stopAuctionBid := range stopAuctionBids {
		instanceGuidsToStop := stopAuctionBid.InstanceGuids
		if stopAuctionBid.Rep == repGuidWithLoneRemainingInstance {
			instanceGuidsToStop = instanceGuidsToStop[1:]
		}
		for _, instanceGuid := range instanceGuidsToStop {
			numCommunication += 1
			wg.Add(1)
			go func(repGuid string, instanceGuid string) {
				client.Stop(repGuid, instanceGuid)
				wg.Done()
			}(stopAuctionBid.Rep, instanceGuid)
		}
	}
	wg.Wait()

	return repGuidWithLoneRemainingInstance, numCommunication, nil
}
