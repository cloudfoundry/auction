package simulation_test

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type AuctionRunnerDelegate struct {
	cells       map[string]auctiontypes.CellRep
	cellLimit   int
	workResults auctiontypes.AuctionResults
	lock        *sync.Mutex
}

func NewAuctionRunnerDelegate(cells map[string]auctiontypes.SimulationCellRep) *AuctionRunnerDelegate {
	typecastCells := map[string]auctiontypes.CellRep{}
	for guid, cell := range cells {
		typecastCells[guid] = cell
	}
	return &AuctionRunnerDelegate{
		cells:     typecastCells,
		cellLimit: len(typecastCells),
		lock:      &sync.Mutex{},
	}
}

func (a *AuctionRunnerDelegate) SetCellLimit(limit int) {
	a.cellLimit = limit
}

func (a *AuctionRunnerDelegate) FetchCellReps() (map[string]auctiontypes.CellRep, error) {
	subset := map[string]auctiontypes.CellRep{}
	for i := 0; i < a.cellLimit; i++ {
		subset[cellGuid(i)] = a.cells[cellGuid(i)]
	}
	return subset, nil
}

func (a *AuctionRunnerDelegate) DistributedBatch(work auctiontypes.AuctionResults) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.workResults.FailedLRPStarts = append(a.workResults.FailedLRPStarts, work.FailedLRPStarts...)
	a.workResults.FailedTasks = append(a.workResults.FailedTasks, work.FailedTasks...)
	a.workResults.SuccessfulLRPStarts = append(a.workResults.SuccessfulLRPStarts, work.SuccessfulLRPStarts...)
	a.workResults.SuccessfulTasks = append(a.workResults.SuccessfulTasks, work.SuccessfulTasks...)
}

func (a *AuctionRunnerDelegate) ResultSize() int {
	a.lock.Lock()
	defer a.lock.Unlock()

	return len(a.workResults.FailedLRPStarts) +
		len(a.workResults.FailedTasks) +
		len(a.workResults.SuccessfulLRPStarts) +
		len(a.workResults.SuccessfulTasks)
}

func (a *AuctionRunnerDelegate) Results() auctiontypes.AuctionResults {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.workResults
}
