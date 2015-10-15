package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/rep"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/pivotal-golang/lager"
)

func FetchStateAndBuildZones(logger lager.Logger, workPool *workpool.WorkPool, clients map[string]rep.Client) map[string]Zone {
	wg := &sync.WaitGroup{}
	zones := map[string]Zone{}
	lock := &sync.Mutex{}

	wg.Add(len(clients))
	for guid, client := range clients {
		guid, client := guid, client
		workPool.Submit(func() {
			defer wg.Done()
			var state rep.CellState
			var err error
			for i := 0; i < 3; i++ {
				state, err = client.State()
				if err != nil && i == 2 {
					logger.Error("failed-to-get-state", err, lager.Data{"cell-guid": guid})
					return
				}
				if err == nil {
					break
				}
			}

			if state.Evacuating {
				return
			}

			cell := NewCell(logger, guid, client, state)
			lock.Lock()
			zones[state.Zone] = append(zones[state.Zone], cell)
			lock.Unlock()
		})
	}

	wg.Wait()

	return zones
}
