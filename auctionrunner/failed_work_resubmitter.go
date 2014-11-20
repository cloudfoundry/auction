package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

func ResubmitFailedWork(batch *Batch, results WorkResults, maxRetries int) WorkResults {
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

	batch.ResubmitStartAuctions(retryableStarts)
	batch.ResubmitStopAuctions(retryableStops)

	return results
}
