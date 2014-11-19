package auctionrunner

import (
	"os"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type AuctionRunner interface {
	AddLRPStartAuction(models.LRPStartAuction)
	AddLRPStopAuction(models.LRPStopAuction)
}

type AuctionRunnerDelegate interface {
	FetchAuctionRepClients() (map[string]auctiontypes.AuctionRep, error)
}

type auctionRunner struct {
	delegate AuctionRunnerDelegate
	hasWork  chan struct{}
}

func New(delegate AuctionRunnerDelegate) *auctionRunner {
	return &auctionRunner{
		delegate: delegate,
		hasWork:  make(chan struct{}, 1),
	}
}

func (a *auctionRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	for {
		select {
		case <-a.hasWork:
			clients, err := a.delegate.FetchAuctionRepClients()
			if err != nil {
				time.Sleep(time.Second)
				a.recordWork()
				continue
			}
			state := a.fetchRepState(clients)
			a.performWork(clients, state)
		case <-signals:
			return nil
		}
	}
}

func (a *auctionRunner) AddLRPStartAuction(models.LRPStartAuction) {
	//record the time, append the start auction
	a.recordWork()
}

func (a *auctionRunner) AddLRPStopAuction(models.LRPStopAuction) {
	//record the time, append the stop auction
	a.recordWork()
}

func (a *auctionRunner) recordWork() {
	select {
	case a.hasWork <- struct{}{}:
	default:
	}
}

func (a *auctionRunner) fetchRepState(clients map[string]auctiontypes.AuctionRep) map[string]auctiontypes.RepState {
	//fetch all the state via a workpool
	//if anything errors, don't include it
	return nil
}

func (a *auctionRunner) performWork(clients map[string]auctiontypes.AuctionRep, state map[string]auctiontypes.RepState) {
	//get the next batch of work (clears out the previous batch)
	//loop through the work (stops first, then starts)
	//on each iteration, figure out where to put the work (pick a winner, keep some sort of state of the winners)
}
