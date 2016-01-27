package auctionrunner

import (
	"sort"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/rep"

	"github.com/cloudfoundry/gunk/workpool"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

type Zone []*Cell

func (z *Zone) FilterCells(rootFS string) []*Cell {
	var cells = make([]*Cell, 0, len(*z))

	for _, cell := range *z {
		if cell.MatchRootFS(rootFS) {
			cells = append(cells, cell)
		}
	}

	return cells
}

type Scheduler struct {
	workPool                *workpool.WorkPool
	zones                   map[string]Zone
	clock                   clock.Clock
	logger                  lager.Logger
	startingContainerWeight float64
}

func NewScheduler(
	workPool *workpool.WorkPool,
	zones map[string]Zone,
	clock clock.Clock,
	logger lager.Logger,
	startingContainerWeight float64,
) *Scheduler {
	return &Scheduler{
		workPool:                workPool,
		zones:                   zones,
		clock:                   clock,
		logger:                  logger,
		startingContainerWeight: startingContainerWeight,
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
		for i, _ := range results.FailedLRPs {
			results.FailedLRPs[i].PlacementError = auctiontypes.ErrorCellCommunication.Error()
		}
		results.FailedTasks = auctionRequest.Tasks
		for i, _ := range results.FailedTasks {
			results.FailedTasks[i].PlacementError = auctiontypes.ErrorCellCommunication.Error()
		}
		return s.markResults(results)
	}

	var successfulLRPs = map[string]*auctiontypes.LRPAuction{}
	var lrpStartAuctionLookup = map[string]*auctiontypes.LRPAuction{}
	var successfulTasks = map[string]*auctiontypes.TaskAuction{}
	var taskAuctionLookup = map[string]*auctiontypes.TaskAuction{}

	sort.Sort(SortableLRPAuctions(auctionRequest.LRPs))
	sort.Sort(SortableTaskAuctions(auctionRequest.Tasks))

	lrpsBeforeTasks, lrpsAfterTasks := splitLRPS(auctionRequest.LRPs)

	auctionLRP := func(lrpsToAuction []auctiontypes.LRPAuction) {
		for i := range lrpsToAuction {
			lrpAuction := &lrpsToAuction[i]
			lrpStartAuctionLookup[lrpAuction.Identifier()] = lrpAuction
			successfulStart, err := s.scheduleLRPAuction(lrpAuction)
			if err != nil {
				lrpAuction.PlacementError = err.Error()
				results.FailedLRPs = append(results.FailedLRPs, *lrpAuction)
			} else {
				successfulLRPs[successfulStart.Identifier()] = successfulStart
			}
		}
	}

	auctionLRP(lrpsBeforeTasks)

	for i := range auctionRequest.Tasks {
		taskAuction := &auctionRequest.Tasks[i]
		taskAuctionLookup[taskAuction.Identifier()] = taskAuction
		successfulTask, err := s.scheduleTaskAuction(taskAuction, s.startingContainerWeight)
		if err != nil {
			taskAuction.PlacementError = err.Error()
			results.FailedTasks = append(results.FailedTasks, *taskAuction)
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

			s.logger.Info("lrp-failed-to-be-placed", lager.Data{"lrp-guid": failedStart.Identifier()})
			results.FailedLRPs = append(results.FailedLRPs, *lrpStartAuctionLookup[identifier])
		}

		for _, failedTask := range failedWork.Tasks {
			identifier := failedTask.Identifier()
			delete(successfulTasks, identifier)

			s.logger.Info("task-failed-to-be-placed", lager.Data{"task-guid": failedTask.Identifier()})
			results.FailedTasks = append(results.FailedTasks, *taskAuctionLookup[identifier])
		}
	}

	for _, successfulStart := range successfulLRPs {
		s.logger.Info("lrp-added-to-cell", lager.Data{"lrp-guid": successfulStart.Identifier(), "cell-guid": successfulStart.Winner})
		results.SuccessfulLRPs = append(results.SuccessfulLRPs, *successfulStart)
	}
	for _, successfulTask := range successfulTasks {
		s.logger.Info("task-added-to-cell", lager.Data{"task-guid": successfulTask.Identifier(), "cell-guid": successfulTask.Winner})
		results.SuccessfulTasks = append(results.SuccessfulTasks, *successfulTask)
	}
	return s.markResults(results)
}

func (s *Scheduler) markResults(results auctiontypes.AuctionResults) auctiontypes.AuctionResults {
	now := s.clock.Now()
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

func (s *Scheduler) commitCells() []rep.Work {
	wg := &sync.WaitGroup{}
	for _, cells := range s.zones {
		wg.Add(len(cells))
	}

	lock := &sync.Mutex{}
	failedWorks := []rep.Work{}

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

func (s *Scheduler) scheduleLRPAuction(lrpAuction *auctiontypes.LRPAuction) (*auctiontypes.LRPAuction, error) {
	var winnerCell *Cell
	winnerScore := 1e20

	zones := accumulateZonesByInstances(s.zones, lrpAuction.ProcessGuid)

	filteredZones := filterZonesByRootFS(zones, lrpAuction.RootFs)

	if len(filteredZones) == 0 {
		return nil, auctiontypes.ErrorCellMismatch
	}

	sortedZones := sortZonesByInstances(filteredZones)

	for zoneIndex, lrpByZone := range sortedZones {
		for _, cell := range lrpByZone.zone {
			score, err := cell.ScoreForLRP(&lrpAuction.LRP, s.startingContainerWeight)
			if err != nil {
				continue
			}

			if score < winnerScore {
				winnerScore = score
				winnerCell = cell
			}
		}

		// if (not last zone) && (this zone has the same # of instances as the next sorted zone)
		// acts as a tie breaker
		if zoneIndex+1 < len(sortedZones) &&
			lrpByZone.instances == sortedZones[zoneIndex+1].instances {
			continue
		}

		if winnerCell != nil {
			break
		}
	}

	if winnerCell == nil {
		return nil, rep.ErrorInsufficientResources
	}

	err := winnerCell.ReserveLRP(&lrpAuction.LRP)
	if err != nil {
		s.logger.Error("lrp-failed-to-reserve-cell", err, lager.Data{"cell-guid": winnerCell.Guid, "lrp-guid": lrpAuction.Identifier()})
		return nil, err
	}

	winningAuction := lrpAuction.Copy()
	winningAuction.Winner = winnerCell.Guid
	return &winningAuction, nil
}

func (s *Scheduler) scheduleTaskAuction(taskAuction *auctiontypes.TaskAuction, startingContainerWeight float64) (*auctiontypes.TaskAuction, error) {
	var winnerCell *Cell
	winnerScore := 1e20

	filteredZones := []Zone{}

	for _, zone := range s.zones {
		cells := zone.FilterCells(taskAuction.RootFs)
		if len(cells) > 0 {
			filteredZones = append(filteredZones, Zone(cells))
		}
	}

	if len(filteredZones) == 0 {
		return nil, auctiontypes.ErrorCellMismatch
	}

	for _, zone := range filteredZones {
		for _, cell := range zone {
			score, err := cell.ScoreForTask(&taskAuction.Task, startingContainerWeight)
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
		return nil, rep.ErrorInsufficientResources
	}

	err := winnerCell.ReserveTask(&taskAuction.Task)
	if err != nil {
		s.logger.Error("task-failed-to-reserve-cell", err, lager.Data{"cell-guid": winnerCell.Guid, "task-guid": taskAuction.Identifier()})
		return nil, err
	}

	winningAuction := taskAuction.Copy()
	winningAuction.Winner = winnerCell.Guid
	return &winningAuction, nil
}
