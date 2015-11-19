package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/cloudfoundry-incubator/rep"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/pivotal-golang/lager"
)

func FetchStateAndBuildZones(logger lager.Logger, workPool *workpool.WorkPool, clients map[string]rep.Client) map[string]Zone {
	var zones map[string]Zone
	for i := 0; ; i++ {
		zones = fetchStateAndBuildZones(logger, workPool, clients)
		if len(zones) > 0 || i == 3 {
			break
		}
		for _, client := range clients {
			stateClient := cf_http.NewCustomTimeoutClient(client.StateClientTimeout() * 2)
			client.SetStateClient(stateClient)
		}
	}
	return zones
}

func fetchStateAndBuildZones(logger lager.Logger, workPool *workpool.WorkPool, clients map[string]rep.Client) map[string]Zone {
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

			cell := NewCell(logger, guid, client, state)
			lock.Lock()
			zones[state.Zone] = append(zones[state.Zone], cell)
			lock.Unlock()
		})
	}

	wg.Wait()

	return zones
}
