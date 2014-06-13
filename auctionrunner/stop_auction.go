package auctionrunner

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

func stopAuction(client auctiontypes.RepPoolClient, auctionRequest auctiontypes.StopAuctionRequest) error {
	stopAuctionInfo := auctiontypes.LRPStopAuctionInfo{
		ProcessGuid: auctionRequest.LRPStopAuction.ProcessGuid,
		Index:       auctionRequest.LRPStopAuction.Index,
	}
	stopScoreResults := client.StopScore(auctionRequest.RepGuids, stopAuctionInfo)
	stopScoreResults = stopScoreResults.FilterErrors()

	instanceGuids := stopScoreResults.InstanceGuids()
	if len(instanceGuids) <= 1 {
		return errors.New("found nothing to stop")
	}

	stopScoreResults = stopScoreResults.Shuffle()

	var repGuidWithLoneRemainingInstance string
	lowestScore := 1e9

	for _, stopScoreResult := range stopScoreResults {
		score := scoreIfRepGuidWins(stopScoreResult)
		if score < lowestScore {
			lowestScore = score
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
			client.Stop(stopScoreResult.Rep, instanceGuid)
		}
	}

	return nil
}

func scoreIfRepGuidWins(stopScoreResult auctiontypes.StopScoreResult) float64 {
	return stopScoreResult.Score - float64(len(stopScoreResult.InstanceGuids)) + 1
}
