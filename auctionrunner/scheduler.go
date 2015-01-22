package auctionrunner

import (
	"sort"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/gunk/workpool"
)

type Scheduler struct {
	workPool     *workpool.WorkPool
	zones        map[string][]*Cell
	timeProvider timeprovider.TimeProvider
}

func NewScheduler(
	workPool *workpool.WorkPool,
	zones map[string][]*Cell,
	timeProvider timeprovider.TimeProvider,
) *Scheduler {
	return &Scheduler{
		workPool:     workPool,
		zones:        zones,
		timeProvider: timeProvider,
	}
}

/*
Schedule takes in a set of job requests (LRP start auctions and task starts) and
assigns the work to available cells according to the diego scoring algorithm. The
scheduler is single-threaded.  It determines scheduling of jobs one at a time so
that each calculation reflects available resources correctly.  It commits the
work in batches at the end, for better network performance.  Schedule returns
AuctionResults, indicating the success or failure of each requested job.
*/
func (s *Scheduler) Schedule(auctionRequest auctiontypes.AuctionRequest) auctiontypes.AuctionResults {
	results := auctiontypes.AuctionResults{}

	if len(s.zones) == 0 {
		results.FailedLRPs = auctionRequest.LRPs
		results.FailedTasks = auctionRequest.Tasks
		return s.markResults(results)
	}

	var successfulLRPs = map[string]auctiontypes.LRPAuction{}
	var lrpStartAuctionLookup = map[string]auctiontypes.LRPAuction{}
	var successfulTasks = map[string]auctiontypes.TaskAuction{}
	var taskAuctionLookup = map[string]auctiontypes.TaskAuction{}

	sort.Sort(SortableLRPAuctions(auctionRequest.LRPs))
	sort.Sort(SortableTaskAuctions(auctionRequest.Tasks))

	lrpsBeforeTasks, lrpsAfterTasks := splitLRPS(auctionRequest.LRPs)

	auctionLRP := func(lrpsToAuction []auctiontypes.LRPAuction) {
		for _, lrpAuction := range lrpsToAuction {
			lrpStartAuctionLookup[lrpAuction.Identifier()] = lrpAuction
			successfulStart, err := s.scheduleLRPAuction(lrpAuction)
			if err != nil {
				lrpAuction.PlacementError = err.Error()
				results.FailedLRPs = append(results.FailedLRPs, lrpAuction)
			} else {
				successfulLRPs[successfulStart.Identifier()] = successfulStart
			}
		}
	}

	auctionLRP(lrpsBeforeTasks)

	for _, taskAuction := range auctionRequest.Tasks {
		taskAuctionLookup[taskAuction.Identifier()] = taskAuction
		successfulTask, err := s.scheduleTaskAuction(taskAuction)
		if err != nil {
			taskAuction.PlacementError = err.Error()
			results.FailedTasks = append(results.FailedTasks, taskAuction)
		} else {
			successfulTasks[successfulTask.Identifier()] = successfulTask
		}
	}

	auctionLRP(lrpsAfterTasks)

	failedWorks := s.commitCells()
	for _, failedWork := range failedWorks {
		for _, failedStart := range failedWork.LRPs {
			identifier := failedStart.Identifier()
			delete(successfulLRPs, identifier)
			results.FailedLRPs = append(results.FailedLRPs, lrpStartAuctionLookup[identifier])
		}

		for _, failedTask := range failedWork.Tasks {
			identifier := auctiontypes.IdentifierForTask(failedTask)
			delete(successfulTasks, identifier)
			results.FailedTasks = append(results.FailedTasks, taskAuctionLookup[identifier])
		}
	}

	for _, successfulStart := range successfulLRPs {
		results.SuccessfulLRPs = append(results.SuccessfulLRPs, successfulStart)
	}
	for _, successfulTask := range successfulTasks {
		results.SuccessfulTasks = append(results.SuccessfulTasks, successfulTask)
	}

	return s.markResults(results)
}

func (s *Scheduler) markResults(results auctiontypes.AuctionResults) auctiontypes.AuctionResults {
	now := s.timeProvider.Now()
	for i := range results.FailedLRPs {

		results.FailedLRPs[i].Attempts++
	}
	for i := range results.FailedTasks {
		results.FailedTasks[i].Attempts++
	}
	for i := range results.SuccessfulLRPs {
		results.SuccessfulLRPs[i].Attempts++
		results.SuccessfulLRPs[i].WaitDuration = now.Sub(results.SuccessfulLRPs[i].QueueTime)
	}
	for i := range results.SuccessfulTasks {
		results.SuccessfulTasks[i].Attempts++
		results.SuccessfulTasks[i].WaitDuration = now.Sub(results.SuccessfulTasks[i].QueueTime)
	}

	return results
}

func splitLRPS(lrps []auctiontypes.LRPAuction) ([]auctiontypes.LRPAuction, []auctiontypes.LRPAuction) {
	const pivot = 0

	for idx, lrp := range lrps {
		if lrp.Index > pivot {
			return lrps[:idx], lrps[idx:]
		}
	}

	return lrps[:0], lrps[0:]
}

func (s *Scheduler) commitCells() []auctiontypes.Work {
	wg := &sync.WaitGroup{}
	for _, cells := range s.zones {
		wg.Add(len(cells))
	}

	lock := &sync.Mutex{}
	failedWorks := []auctiontypes.Work{}

	for _, cells := range s.zones {
		for _, cell := range cells {
			cell := cell
			s.workPool.Submit(func() {
				defer wg.Done()
				failedWork := cell.Commit()

				lock.Lock()
				failedWorks = append(failedWorks, failedWork)
				lock.Unlock()
			})
		}
	}

	wg.Wait()
	return failedWorks
}

func (s *Scheduler) scheduleLRPAuction(lrpAuction auctiontypes.LRPAuction) (auctiontypes.LRPAuction, error) {
	var winnerCell *Cell
	winnerScore := 1e20

	sortedZones := sortZonesByInstances(s.zones, lrpAuction)

	for zoneIndex, zone := range sortedZones {
		for _, cell := range zone.cells {
			score, err := cell.ScoreForLRPAuction(lrpAuction)
			if err != nil {
				continue
			}

			if score < winnerScore {
				winnerScore = score
				winnerCell = cell
			}
		}

		if zoneIndex+1 < len(sortedZones) &&
			zone.instances == sortedZones[zoneIndex+1].instances {
			continue
		}

		if winnerCell != nil {
			break
		}
	}

	if winnerCell == nil {
		return auctiontypes.LRPAuction{}, auctiontypes.ErrorInsufficientResources
	}

	err := winnerCell.ReserveLRP(lrpAuction)
	if err != nil {
		return auctiontypes.LRPAuction{}, err
	}

	lrpAuction.Winner = winnerCell.Guid
	return lrpAuction, nil
}

func (s *Scheduler) scheduleTaskAuction(taskAuction auctiontypes.TaskAuction) (auctiontypes.TaskAuction, error) {
	var winnerCell *Cell
	winnerScore := 1e20

	for _, cells := range s.zones {
		for _, cell := range cells {
			score, err := cell.ScoreForTask(taskAuction.Task)
			if err != nil {
				continue
			}

			if score < winnerScore {
				winnerScore = score
				winnerCell = cell
			}
		}
	}

	if winnerCell == nil {
		return auctiontypes.TaskAuction{}, auctiontypes.ErrorInsufficientResources
	}

	err := winnerCell.ReserveTask(taskAuction.Task)
	if err != nil {
		return auctiontypes.TaskAuction{}, err
	}

	taskAuction.Winner = winnerCell.Guid
	return taskAuction, nil
}
