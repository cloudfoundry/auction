package auctionrunner

import (
	"math/rand"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/gunk/workpool"
)

func DistributeWork(workPool *workpool.WorkPool, cells map[string]*Cell, timeProvider timeprovider.TimeProvider, startAuctions []auctiontypes.StartAuction, stopAuctions []auctiontypes.StopAuction) WorkResults {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	results := WorkResults{}
	if len(cells) == 0 {
		results.FailedStarts = startAuctions
		results.FailedStops = stopAuctions
		return markResults(results, timeProvider)
	}

	for _, stopAuction := range stopAuctions {
		succesfulStop := processStopAuction(cells, stopAuction)
		results.SuccessfulStops = append(results.SuccessfulStops, succesfulStop)
	}

	var successfulStarts = map[string]auctiontypes.StartAuction{}
	var startAuctionLookup = map[string]auctiontypes.StartAuction{}

	for _, startAuction := range startAuctions {
		startAuctionLookup[startAuction.Identifier()] = startAuction

		successfulStart, err := processStartAuction(cells, startAuction, randomizer)
		if err != nil {
			results.FailedStarts = append(results.FailedStarts, startAuction)
			continue
		}
		successfulStarts[successfulStart.Identifier()] = successfulStart
	}

	failedWorks := commitCells(workPool, cells)
	for _, failedWork := range failedWorks {
		for _, failedStart := range failedWork.Starts {
			identifier := auctiontypes.IdentifierForLRPStartAuction(failedStart)
			delete(successfulStarts, identifier)
			results.FailedStarts = append(results.FailedStarts, startAuctionLookup[identifier])
		}
	}

	for _, successfulStart := range successfulStarts {
		results.SuccessfulStarts = append(results.SuccessfulStarts, successfulStart)
	}

	return markResults(results, timeProvider)
}

func markResults(results WorkResults, timeProvider timeprovider.TimeProvider) WorkResults {
	now := timeProvider.Time()
	for i := range results.FailedStarts {
		results.FailedStarts[i].Attempts++
	}
	for i := range results.FailedStops {
		results.FailedStops[i].Attempts++
	}
	for i := range results.SuccessfulStarts {
		results.SuccessfulStarts[i].Attempts++
		results.SuccessfulStarts[i].WaitDuration = now.Sub(results.SuccessfulStarts[i].QueueTime)
	}
	for i := range results.SuccessfulStops {
		results.SuccessfulStops[i].Attempts++
		results.SuccessfulStops[i].WaitDuration = now.Sub(results.SuccessfulStops[i].QueueTime)
	}

	return results
}

func commitCells(workPool *workpool.WorkPool, cells map[string]*Cell) []auctiontypes.Work {
	wg := &sync.WaitGroup{}
	wg.Add(len(cells))

	lock := &sync.Mutex{}
	failedWorks := []auctiontypes.Work{}

	for _, cell := range cells {
		cell := cell
		workPool.Submit(func() {
			failedWork := cell.Commit()

			lock.Lock()
			failedWorks = append(failedWorks, failedWork)
			lock.Unlock()

			wg.Done()
		})
	}

	wg.Wait()
	return failedWorks
}

func processStartAuction(cells map[string]*Cell, startAuction auctiontypes.StartAuction, randomizer *rand.Rand) (auctiontypes.StartAuction, error) {
	winnerGuids := []string{}
	winnerScore := 1e20

	for guid, cell := range cells {
		score, err := cell.ScoreForStartAuction(startAuction.LRPStartAuction)
		if err != nil {
			continue
		}

		if score == winnerScore {
			winnerGuids = append(winnerGuids, guid)
		} else if score < winnerScore {
			winnerScore = score
			winnerGuids = []string{guid}
		}
	}

	if len(winnerGuids) == 0 {
		return auctiontypes.StartAuction{}, ErrorInsufficientResources
	}

	winnerGuid := winnerGuids[randomizer.Intn(len(winnerGuids))]

	err := cells[winnerGuid].StartLRP(startAuction.LRPStartAuction)
	if err != nil {
		return auctiontypes.StartAuction{}, err
	}

	startAuction.Winner = winnerGuid

	return startAuction, nil
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
