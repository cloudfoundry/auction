package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry/gunk/workpool"
)

func FetchStateAndBuildCells(workPool *workpool.WorkPool, clients map[string]auctiontypes.CellRep) map[string]*Cell {
	wg := &sync.WaitGroup{}
	cells := map[string]*Cell{}
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
			cell := NewCell(client, state)
			lock.Lock()
			cells[guid] = cell
			lock.Unlock()
		})
	}

	wg.Wait()

	return cells
}
