package auctionrunner

import (
	"fmt"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/gunk/workpool"
)

type DistributeWorkResults struct {
	SuccessfulStarts []auctiontypes.StartAuction
	SuccessfulStops  []auctiontypes.StopAuction
	FailedStarts     []auctiontypes.StartAuction
	FailedStops      []auctiontypes.StopAuction
}

func DistributeWork(workPool *workpool.WorkPool, cells map[string]*Cell, timeProvider timeprovider.TimeProvider, startAuctions []auctiontypes.StartAuction, stopAuctions []auctiontypes.StopAuction) DistributeWorkResults {
	results := DistributeWorkResults{}
	if len(cells) == 0 {
		markStartsAsFailed(startAuctions)
		markStopsAsFailed(stopAuctions)
		results.FailedStarts = startAuctions
		results.FailedStops = stopAuctions
		return results
	}

	for _, stopAuction := range stopAuctions {
		succesfulStop := processStopAuction(cells, stopAuction)
		results.SuccessfulStops = append(results.SuccessfulStops, succesfulStop)
	}

	failedWork := commitCells(workPool, cells)
	fmt.Println("deal with", failedWork)

	markStopsAsAsSucceeded(results.SuccessfulStops, timeProvider)

	return results
}

func markStartsAsFailed(startAuctions []auctiontypes.StartAuction) {
	for i := range startAuctions {
		startAuctions[i].Attempts++
	}
}

func markStopsAsFailed(stopAuctions []auctiontypes.StopAuction) {
	for i := range stopAuctions {
		stopAuctions[i].Attempts++
	}
}

func markStopsAsAsSucceeded(stopAuctions []auctiontypes.StopAuction, timeProvider timeprovider.TimeProvider) {
	now := timeProvider.Time()
	for i := range stopAuctions {
		stopAuctions[i].Attempts++
		stopAuctions[i].WaitDuration = now.Sub(stopAuctions[i].QueueTime)
	}
}

func commitCells(workPool *workpool.WorkPool, cells map[string]*Cell) []auctiontypes.Work {
	wg := &sync.WaitGroup{}
	wg.Add(len(cells))

	lock := &sync.Mutex{}
	failedWork := []auctiontypes.Work{}

	for _, cell := range cells {
		cell := cell
		workPool.Submit(func() {
			failedWorkOnCell := cell.Commit()

			lock.Lock()
			failedWork = append(failedWork, failedWorkOnCell)
			lock.Unlock()

			wg.Done()
		})
	}

	wg.Wait()
	return failedWork
}

func processStopAuction(cells map[string]*Cell, stopAuction auctiontypes.StopAuction) auctiontypes.StopAuction {
	winnerGuid := ""
	winnerScore := 1e20
	instancesToStop := map[string][]string{}

	for guid, cell := range cells {
		score, instances, err := cell.ScoreForStopAuction(stopAuction.LRPStopAuction)
		if err != nil {
			continue
		}

		instancesToStop[guid] = instances

		if score < winnerScore {
			winnerGuid = guid
			winnerScore = score
		}
	}

	if len(instancesToStop) == 0 {
		//no one's got this instance, we're done.  if it's still out there we'll eventually try again.
		return stopAuction
	}

	stopAuction.Winner = winnerGuid

	if len(instancesToStop[winnerGuid]) > 1 {
		for _, instance := range instancesToStop[winnerGuid][1:] {
			cells[winnerGuid].StopLRP(models.StopLRPInstance{
				ProcessGuid:  stopAuction.LRPStopAuction.ProcessGuid,
				InstanceGuid: instance,
				Index:        stopAuction.LRPStopAuction.Index,
			})
		}
	}

	delete(instancesToStop, winnerGuid)

	for guid, instances := range instancesToStop {
		for _, instance := range instances {
			cells[guid].StopLRP(models.StopLRPInstance{
				ProcessGuid:  stopAuction.LRPStopAuction.ProcessGuid,
				InstanceGuid: instance,
				Index:        stopAuction.LRPStopAuction.Index,
			})
		}
	}

	return stopAuction
}
