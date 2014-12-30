package auctionrunner

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auctioneer"
)

func ResubmitFailedAuctions(batch *Batch, results auctiontypes.AuctionResults, maxRetries int) auctiontypes.AuctionResults {
	retryableLRPs := []auctiontypes.LRPAuction{}
	retryableTasks := []auctiontypes.TaskAuction{}
	failedLRPs := []auctiontypes.LRPAuction{}
	failedTasks := []auctiontypes.TaskAuction{}

	for _, start := range results.FailedLRPs {
		if start.Attempts <= maxRetries {
			retryableLRPs = append(retryableLRPs, start)
		} else {
			failedLRPs = append(failedLRPs, start)
		}
	}

	for _, task := range results.FailedTasks {
		if task.Attempts <= maxRetries {
			retryableTasks = append(retryableTasks, task)
		} else {
			failedTasks = append(failedTasks, task)
		}
	}

	if len(retryableLRPs) > 0 {
		batch.ResubmitStartAuctions(retryableLRPs)
	}
	if len(retryableTasks) > 0 {
		batch.ResubmitTaskAuctions(retryableTasks)
	}

	auctioneer.LRPAuctionsFailed.Add(uint64(len(failedLRPs)))
	auctioneer.TaskAuctionsFailed.Add(uint64(len(failedTasks)))

	results.FailedLRPs = failedLRPs
	results.FailedTasks = failedTasks

	return results
}
