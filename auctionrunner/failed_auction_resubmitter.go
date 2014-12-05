package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

func ResubmitFailedAuctions(batch *Batch, results auctiontypes.AuctionResults, maxRetries int) auctiontypes.AuctionResults {
	retryableLRPStarts := []auctiontypes.LRPStartAuction{}
	retryableLRPStops := []auctiontypes.LRPStopAuction{}
	failedLRPStarts := []auctiontypes.LRPStartAuction{}
	failedLRPStops := []auctiontypes.LRPStopAuction{}

	for _, start := range results.FailedLRPStarts {
		if start.Attempts <= maxRetries {
			retryableLRPStarts = append(retryableLRPStarts, start)
		} else {
			failedLRPStarts = append(failedLRPStarts, start)
		}
	}

	for _, stop := range results.FailedLRPStops {
		if stop.Attempts <= maxRetries {
			retryableLRPStops = append(retryableLRPStops, stop)
		} else {
			failedLRPStops = append(failedLRPStops, stop)
		}
	}

	results.FailedLRPStarts = failedLRPStarts
	results.FailedLRPStops = failedLRPStops

	if len(retryableLRPStarts) > 0 {
		batch.ResubmitStartAuctions(retryableLRPStarts)
	}
	if len(retryableLRPStops) > 0 {
		batch.ResubmitStopAuctions(retryableLRPStops)
	}

	return results
}
