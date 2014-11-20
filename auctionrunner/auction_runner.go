package auctionrunner

import (
	"os"
	"time"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/pivotal-golang/lager"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
)

type auctionRunner struct {
	delegate     auctiontypes.AuctionRunnerDelegate
	batch        *Batch
	timeProvider timeprovider.TimeProvider
	workPool     *workpool.WorkPool
	maxRetries   int
	logger       lager.Logger
}

func New(delegate auctiontypes.AuctionRunnerDelegate, timeProvider timeprovider.TimeProvider, maxRetries int, workPool *workpool.WorkPool, logger lager.Logger) *auctionRunner {
	return &auctionRunner{
		delegate:     delegate,
		batch:        NewBatch(timeProvider),
		timeProvider: timeProvider,
		workPool:     workPool,
		maxRetries:   maxRetries,
		logger:       logger,
	}
}

func (a *auctionRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	var hasWork chan struct{}
	hasWork = a.batch.HasWork

	for {
		select {
		case <-hasWork:
			logger := a.logger.Session("auction")

			logger.Info("fetching-cell-reps")
			clients, err := a.delegate.FetchCellReps()
			if err != nil {
				logger.Error("failed-to-fetch-reps", err)
				time.Sleep(time.Second)
				hasWork = make(chan struct{}, 1)
				hasWork <- struct{}{}
				break
			}
			logger.Info("fetched-cell-reps", lager.Data{"cell-reps-count": len(clients)})

			hasWork = a.batch.HasWork

			logger.Info("fetching-cell-state")
			cells := FetchStateAndBuildCells(a.workPool, clients)
			logger.Info("fetched-cell-state", lager.Data{"cell-state-count": len(cells), "num-failed-requests": len(clients) - len(cells)})

			logger.Info("fetching-work")
			startAuctions, stopAuctions := a.batch.DedupeAndDrain()
			logger.Info("fetched-work", lager.Data{"start-auctions": len(startAuctions), "stop-auctions": len(stopAuctions)})
			if len(startAuctions) == 0 && len(stopAuctions) == 0 {
				logger.Info("no-work-to-auction")
				break
			}

			logger.Info("distributing-work")
			workResults := DistributeWork(a.workPool, cells, a.timeProvider, startAuctions, stopAuctions)
			logger.Info("distributed-work", lager.Data{
				"successful-start-auctions": len(workResults.SuccessfulStarts),
				"successful-stop-auctions":  len(workResults.SuccessfulStops),
				"failed-start-auctions":     len(workResults.FailedStarts),
				"failed-stop-auctions":      len(workResults.FailedStops),
			})
			numStartsFailed := len(workResults.FailedStarts)
			numStopsFailed := len(workResults.FailedStops)

			logger.Info("resubmitting-failures")
			workResults = ResubmitFailedWork(a.batch, workResults, a.maxRetries)
			logger.Info("resubmitted-failures", lager.Data{
				"successful-start-auctions":     len(workResults.SuccessfulStarts),
				"successful-stop-auctions":      len(workResults.SuccessfulStops),
				"will-not-retry-start-auctions": len(workResults.FailedStarts),
				"will-not-retry-stop-auctions":  len(workResults.FailedStops),
				"will-retry-start-auctions":     numStartsFailed - len(workResults.FailedStarts),
				"will-retry-stop-auctions":      numStopsFailed - len(workResults.FailedStops),
			})

			a.delegate.DistributedBatch(workResults)
		case <-signals:
			return nil
		}
	}
}

func (a *auctionRunner) AddLRPStartAuction(start models.LRPStartAuction) {
	a.batch.AddLRPStartAuction(start)
}

func (a *auctionRunner) AddLRPStopAuction(stop models.LRPStopAuction) {
	a.batch.AddLRPStopAuction(stop)
}
