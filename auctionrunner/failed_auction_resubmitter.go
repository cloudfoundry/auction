package auctionrunner

import "github.com/cloudfoundry-incubator/auction/auctiontypes"

func ResubmitFailedAuctions(batch *Batch, results auctiontypes.AuctionResults, maxRetries int) auctiontypes.AuctionResults {
	retryableLRPStarts := []auctiontypes.LRPStartAuction{}
	retryableTasks := []auctiontypes.TaskAuction{}
	failedLRPStarts := []auctiontypes.LRPStartAuction{}
	failedTasks := []auctiontypes.TaskAuction{}

	for _, start := range results.FailedLRPStarts {
		if start.Attempts <= maxRetries {
			retryableLRPStarts = append(retryableLRPStarts, start)
		} else {
			failedLRPStarts = append(failedLRPStarts, start)
		}
	}

	for _, task := range results.FailedTasks {
		if task.Attempts <= maxRetries {
			retryableTasks = append(retryableTasks, task)
		} else {
			failedTasks = append(failedTasks, task)
		}
	}

	results.FailedLRPStarts = failedLRPStarts
	results.FailedTasks = failedTasks

	if len(retryableLRPStarts) > 0 {
		batch.ResubmitStartAuctions(retryableLRPStarts)
	}
	if len(retryableTasks) > 0 {
		batch.ResubmitTaskAuctions(retryableTasks)
	}
	return results
}
