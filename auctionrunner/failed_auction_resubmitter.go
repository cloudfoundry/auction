package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

func ResubmitFailedAuctions(batch *Batch, results auctiontypes.AuctionResults, maxRetries int) auctiontypes.AuctionResults {
	retryableStarts := []auctiontypes.StartAuction{}
	retryableStops := []auctiontypes.StopAuction{}
	failedStarts := []auctiontypes.StartAuction{}
	failedStops := []auctiontypes.StopAuction{}

	for _, start := range results.FailedStarts {
		if start.Attempts <= maxRetries {
			retryableStarts = append(retryableStarts, start)
		} else {
			failedStarts = append(failedStarts, start)
		}
	}

	for _, stop := range results.FailedStops {
		if stop.Attempts <= maxRetries {
			retryableStops = append(retryableStops, stop)
		} else {
			failedStops = append(failedStops, stop)
		}
	}

	results.FailedStarts = failedStarts
	results.FailedStops = failedStops

	if len(retryableStarts) > 0 {
		batch.ResubmitStartAuctions(retryableStarts)
	}
	if len(retryableStops) > 0 {
		batch.ResubmitStopAuctions(retryableStops)
	}

	return results
}
