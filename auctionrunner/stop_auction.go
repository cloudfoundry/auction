package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

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

	for _, stopAuctionBid := range stopAuctionBids {
		instanceGuidsToStop := stopAuctionBid.InstanceGuids
		if stopAuctionBid.Rep == repGuidWithLoneRemainingInstance {
			instanceGuidsToStop = instanceGuidsToStop[1:]
		}
		for _, instanceGuid := range instanceGuidsToStop {
			//this is terrible
			numCommunication += 1
			client.Stop(stopAuctionBid.Rep, instanceGuid)
		}
	}

	return repGuidWithLoneRemainingInstance, numCommunication, nil
}
