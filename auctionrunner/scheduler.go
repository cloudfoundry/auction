package auctionrunner

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/gunk/workpool"
)

func Schedule(workPool *workpool.WorkPool, cells map[string]*Cell, timeProvider timeprovider.TimeProvider, lrpStartAuctions []auctiontypes.LRPStartAuction, lrpStopAuctions []auctiontypes.LRPStopAuction) auctiontypes.AuctionResults {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	results := auctiontypes.AuctionResults{}
	if len(cells) == 0 {
		results.FailedLRPStarts = lrpStartAuctions
		results.FailedLRPStops = lrpStopAuctions
		return markResults(results, timeProvider)
	}

	for _, stopAuction := range lrpStopAuctions {
		succesfulStop := scheduleLRPStopAuction(cells, stopAuction)
		results.SuccessfulLRPStops = append(results.SuccessfulLRPStops, succesfulStop)
	}
	var successfulLRPStarts = map[string]auctiontypes.LRPStartAuction{}
	var lrpStartAuctionLookup = map[string]auctiontypes.LRPStartAuction{}

	sort.Sort(sort.Reverse(SortableAuctions(lrpStartAuctions)))

	for _, startAuction := range lrpStartAuctions {
		lrpStartAuctionLookup[startAuction.Identifier()] = startAuction

		successfulStart, err := scheduleLRPStartAuction(cells, startAuction, randomizer)
		if err != nil {
			results.FailedLRPStarts = append(results.FailedLRPStarts, startAuction)
			continue
		}
		successfulLRPStarts[successfulStart.Identifier()] = successfulStart
	}

	failedWorks := commitCells(workPool, cells)
	for _, failedWork := range failedWorks {
		for _, failedStart := range failedWork.LRPStarts {
			identifier := auctiontypes.IdentifierForLRPStartAuction(failedStart)
			delete(successfulLRPStarts, identifier)
			results.FailedLRPStarts = append(results.FailedLRPStarts, lrpStartAuctionLookup[identifier])
		}
	}

	for _, successfulStart := range successfulLRPStarts {
		results.SuccessfulLRPStarts = append(results.SuccessfulLRPStarts, successfulStart)
	}

	return markResults(results, timeProvider)
}

func markResults(results auctiontypes.AuctionResults, timeProvider timeprovider.TimeProvider) auctiontypes.AuctionResults {
	now := timeProvider.Now()
	for i := range results.FailedLRPStarts {
		results.FailedLRPStarts[i].Attempts++
	}
	for i := range results.FailedLRPStops {
		results.FailedLRPStops[i].Attempts++
	}
	for i := range results.SuccessfulLRPStarts {
		results.SuccessfulLRPStarts[i].Attempts++
		results.SuccessfulLRPStarts[i].WaitDuration = now.Sub(results.SuccessfulLRPStarts[i].QueueTime)
	}
	for i := range results.SuccessfulLRPStops {
		results.SuccessfulLRPStops[i].Attempts++
		results.SuccessfulLRPStops[i].WaitDuration = now.Sub(results.SuccessfulLRPStops[i].QueueTime)
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

func scheduleLRPStartAuction(cells map[string]*Cell, lrpStartAuction auctiontypes.LRPStartAuction, randomizer *rand.Rand) (auctiontypes.LRPStartAuction, error) {
	winnerGuids := []string{}
	winnerScore := 1e20

	for guid, cell := range cells {
		score, err := cell.ScoreForLRPStartAuction(lrpStartAuction.LRPStartAuction)
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
		return auctiontypes.LRPStartAuction{}, auctiontypes.ErrorInsufficientResources
	}

	winnerGuid := winnerGuids[randomizer.Intn(len(winnerGuids))]

	err := cells[winnerGuid].StartLRP(lrpStartAuction.LRPStartAuction)
	if err != nil {
		return auctiontypes.LRPStartAuction{}, err
	}

	lrpStartAuction.Winner = winnerGuid

	return lrpStartAuction, nil
}

func scheduleLRPStopAuction(cells map[string]*Cell, lrpStopAuction auctiontypes.LRPStopAuction) auctiontypes.LRPStopAuction {
	winnerGuid := ""
	winnerScore := 1e20
	instancesToStop := map[string][]string{}

	for guid, cell := range cells {
		score, instances, err := cell.ScoreForLRPStopAuction(lrpStopAuction.LRPStopAuction)
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
		return lrpStopAuction
	}

	lrpStopAuction.Winner = winnerGuid

	if len(instancesToStop[winnerGuid]) > 1 {
		for _, instance := range instancesToStop[winnerGuid][1:] {
			cells[winnerGuid].StopLRP(models.ActualLRP{
				ProcessGuid:  lrpStopAuction.LRPStopAuction.ProcessGuid,
				InstanceGuid: instance,
				Index:        lrpStopAuction.LRPStopAuction.Index,
				CellID:       winnerGuid,
			})
		}
	}

	delete(instancesToStop, winnerGuid)

	for guid, instances := range instancesToStop {
		for _, instance := range instances {
			cells[guid].StopLRP(models.ActualLRP{
				ProcessGuid:  lrpStopAuction.LRPStopAuction.ProcessGuid,
				InstanceGuid: instance,
				Index:        lrpStopAuction.LRPStopAuction.Index,
				CellID:       guid,
			})
		}
	}

	return lrpStopAuction
}

type SortableAuctions []auctiontypes.LRPStartAuction

func (a SortableAuctions) Len() int      { return len(a) }
func (a SortableAuctions) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortableAuctions) Less(i, j int) bool {
	return a[i].LRPStartAuction.DesiredLRP.MemoryMB < a[j].LRPStartAuction.DesiredLRP.MemoryMB
}
