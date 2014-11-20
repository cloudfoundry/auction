package simulation_test

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type AuctionRunnerDelegate struct {
	cells       map[string]auctiontypes.AuctionRep
	cellLimit   int
	workResults auctionrunner.WorkResults
	lock        *sync.Mutex
}

func NewAuctionRunnerDelegate(cells map[string]auctiontypes.SimulationAuctionRep) *AuctionRunnerDelegate {
	typecastCells := map[string]auctiontypes.AuctionRep{}
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

func (a *AuctionRunnerDelegate) FetchAuctionRepClients() (map[string]auctiontypes.AuctionRep, error) {
	subset := map[string]auctiontypes.AuctionRep{}
	for i := 0; i < a.cellLimit; i++ {
		subset[cellGuid(i)] = a.cells[cellGuid(i)]
	}
	return subset, nil
}

func (a *AuctionRunnerDelegate) DistributedBatch(work auctionrunner.WorkResults) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.workResults.FailedStarts = append(a.workResults.FailedStarts, work.FailedStarts...)
	a.workResults.FailedStops = append(a.workResults.FailedStops, work.FailedStops...)
	a.workResults.SuccessfulStarts = append(a.workResults.SuccessfulStarts, work.SuccessfulStarts...)
	a.workResults.SuccessfulStops = append(a.workResults.SuccessfulStops, work.SuccessfulStops...)
}

func (a *AuctionRunnerDelegate) ResultSize() int {
	a.lock.Lock()
	defer a.lock.Unlock()

	return len(a.workResults.FailedStarts) +
		len(a.workResults.FailedStops) +
		len(a.workResults.SuccessfulStarts) +
		len(a.workResults.SuccessfulStops)
}

func (a *AuctionRunnerDelegate) Results() auctionrunner.WorkResults {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.workResults
}
