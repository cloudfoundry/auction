package auctionrunner

import (
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/gunk/workpool"
)

/*
Schedule takes in a set of job requests (LRP start auctions and task starts) and
assigns the work to available cells according to the diego scoring algorithm. The
scheduler is single-threaded.  It determines scheduling of jobs one at a time so
that each calculation reflects available resources correctly.  It commits the
work in batches at the end, for better network performance.  Schedule returns
AuctionResults, indicating the success or failure of each requested job.
*/
func Schedule(workPool *workpool.WorkPool, cells map[string]*Cell, timeProvider timeprovider.TimeProvider, auctionRequest auctiontypes.AuctionRequest) auctiontypes.AuctionResults {
	randomizer := rand.New(rand.NewSource(time.Now().UnixNano()))

	results := auctiontypes.AuctionResults{}

	if len(cells) == 0 {
		results.FailedLRPStarts = auctionRequest.LRPStarts
		results.FailedTasks = auctionRequest.Tasks
		return markResults(results, timeProvider)
	}

	var successfulLRPStarts = map[string]auctiontypes.LRPStartAuction{}
	var lrpStartAuctionLookup = map[string]auctiontypes.LRPStartAuction{}

	sort.Sort(sort.Reverse(SortableAuctions(auctionRequest.LRPStarts)))
	for _, startAuction := range auctionRequest.LRPStarts {
		lrpStartAuctionLookup[startAuction.Identifier()] = startAuction

		successfulStart, err := scheduleLRPStartAuction(cells, startAuction, randomizer)
		if err != nil {
			results.FailedLRPStarts = append(results.FailedLRPStarts, startAuction)
		} else {
			successfulLRPStarts[successfulStart.Identifier()] = successfulStart
		}
	}

	var successfulTasks = map[string]auctiontypes.TaskAuction{}
	var taskAuctionLookup = map[string]auctiontypes.TaskAuction{}
	for _, taskAuction := range auctionRequest.Tasks {
		taskAuctionLookup[taskAuction.Identifier()] = taskAuction
		successfulTask, err := scheduleTaskAuction(cells, taskAuction, randomizer)
		if err != nil {
			results.FailedTasks = append(results.FailedTasks, taskAuction)
		} else {
			successfulTasks[successfulTask.Identifier()] = successfulTask
		}
	}

	failedWorks := commitCells(workPool, cells)
	for _, failedWork := range failedWorks {
		for _, failedStart := range failedWork.LRPStarts {
			identifier := auctiontypes.IdentifierForLRPStartAuction(failedStart)
			delete(successfulLRPStarts, identifier)
			results.FailedLRPStarts = append(results.FailedLRPStarts, lrpStartAuctionLookup[identifier])
		}

		for _, failedTask := range failedWork.Tasks {
			identifier := auctiontypes.IdentifierForTask(failedTask)
			delete(successfulTasks, identifier)
			results.FailedTasks = append(results.FailedTasks, taskAuctionLookup[identifier])
		}
	}

	for _, successfulStart := range successfulLRPStarts {
		results.SuccessfulLRPStarts = append(results.SuccessfulLRPStarts, successfulStart)
	}
	for _, successfulTask := range successfulTasks {
		results.SuccessfulTasks = append(results.SuccessfulTasks, successfulTask)
	}

	return markResults(results, timeProvider)
}

func markResults(results auctiontypes.AuctionResults, timeProvider timeprovider.TimeProvider) auctiontypes.AuctionResults {
	now := timeProvider.Now()
	for i := range results.FailedLRPStarts {
		results.FailedLRPStarts[i].Attempts++
	}
	for i := range results.FailedTasks {
		results.FailedTasks[i].Attempts++
	}
	for i := range results.SuccessfulLRPStarts {
		results.SuccessfulLRPStarts[i].Attempts++
		results.SuccessfulLRPStarts[i].WaitDuration = now.Sub(results.SuccessfulLRPStarts[i].QueueTime)
	}
	for i := range results.SuccessfulTasks {
		results.SuccessfulTasks[i].Attempts++
		results.SuccessfulTasks[i].WaitDuration = now.Sub(results.SuccessfulTasks[i].QueueTime)
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

func scheduleTaskAuction(cells map[string]*Cell, taskAuction auctiontypes.TaskAuction, randomizer *rand.Rand) (auctiontypes.TaskAuction, error) {
	winnerGuids := []string{}
	winnerScore := 1e20

	for guid, cell := range cells {
		score, err := cell.ScoreForTask(taskAuction.Task)
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
		return auctiontypes.TaskAuction{}, auctiontypes.ErrorInsufficientResources
	}

	winnerGuid := winnerGuids[randomizer.Intn(len(winnerGuids))]

	err := cells[winnerGuid].StartTask(taskAuction.Task)
	if err != nil {
		return auctiontypes.TaskAuction{}, err
	}

	taskAuction.Winner = winnerGuid

	return taskAuction, nil
}

type SortableAuctions []auctiontypes.LRPStartAuction

func (a SortableAuctions) Len() int      { return len(a) }
func (a SortableAuctions) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortableAuctions) Less(i, j int) bool {
	return a[i].LRPStartAuction.DesiredLRP.MemoryMB < a[j].LRPStartAuction.DesiredLRP.MemoryMB
}
