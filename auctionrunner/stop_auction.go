package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

func stopAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StopAuctionRequest) (string, int, error) {
	numCommunication := 0

	stopAuctionInfo := auctiontypes.LRPStopAuctionInfo{
		ProcessGuid: auctionRequest.LRPStopAuction.ProcessGuid,
		Index:       auctionRequest.LRPStopAuction.Index,
	}

	numCommunication += len(auctionRequest.RepGuids)
	stopScoreResults := client.StopScore(auctionRequest.RepGuids, stopAuctionInfo)
	stopScoreResults = stopScoreResults.FilterErrors()

	instanceGuids := stopScoreResults.InstanceGuids()
	if len(instanceGuids) <= 1 {
		return "", numCommunication, auctiontypes.NothingToStop
	}

	stopScoreResults = stopScoreResults.Shuffle()

	var repGuidWithLoneRemainingInstance string
	lowestScore := 1e9

	for _, stopScoreResult := range stopScoreResults {
		scoreIfRepGuidWins := stopScoreResult.Score - float64(len(stopScoreResult.InstanceGuids)) + 1
		if scoreIfRepGuidWins < lowestScore {
			lowestScore = scoreIfRepGuidWins
			repGuidWithLoneRemainingInstance = stopScoreResult.Rep
		}
	}

	for _, stopScoreResult := range stopScoreResults {
		instanceGuidsToStop := stopScoreResult.InstanceGuids
		if stopScoreResult.Rep == repGuidWithLoneRemainingInstance {
			instanceGuidsToStop = instanceGuidsToStop[1:]
		}
		for _, instanceGuid := range instanceGuidsToStop {
			//this is terrible
			numCommunication += 1
			client.Stop(stopScoreResult.Rep, instanceGuid)
		}
	}

	return repGuidWithLoneRemainingInstance, numCommunication, nil
}
