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
			state, err := client.State()
			if err != nil {
				logger.Error("failed-to-get-state", err, lager.Data{"cell-guid": guid})
				return
			}

			if state.Evacuating {
				return
			}

			cell := NewCell(guid, client, state)
			lock.Lock()
			zones[state.Zone] = append(zones[state.Zone], cell)
			lock.Unlock()
		})
	}

	wg.Wait()

	return zones
}
