package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry/gunk/workpool"
)

func FetchStateAndBuildZones(workPool *workpool.WorkPool, clients map[string]auctiontypes.CellRep) map[string][]*Cell {
	wg := &sync.WaitGroup{}
	zones := map[string][]*Cell{}
	lock := &sync.Mutex{}

	wg.Add(len(clients))
	for guid, client := range clients {
		guid, client := guid, client
		workPool.Submit(func() {
			defer wg.Done()
			state, err := client.State()
			if err != nil {
				return
			}
			cell := NewCell(guid, client, state)
			lock.Lock()
			cells, found := zones[state.Zone]
			if found {
				cells = append(cells, cell)
			} else {
				cells = []*Cell{cell}
			}
			zones[state.Zone] = cells
			lock.Unlock()
		})
	}

	wg.Wait()

	return zones
}
