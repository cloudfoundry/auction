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

			logger.Info("fetching-auctions")
			lrpStartAuctions, lrpStopAuctions, _ := a.batch.DedupeAndDrain()
			logger.Info("fetched-auctions", lager.Data{"lrp-start-auctions": len(lrpStartAuctions), "lrp-stop-auctions": len(lrpStopAuctions)})
			if len(lrpStartAuctions) == 0 && len(lrpStopAuctions) == 0 {
				logger.Info("nothing-to-auction")
				break
			}

			logger.Info("scheduling")
			auctionResults := Schedule(a.workPool, cells, a.timeProvider, lrpStartAuctions, lrpStopAuctions)
			logger.Info("scheduled", lager.Data{
				"successful-lrp-start-auctions": len(auctionResults.SuccessfulLRPStarts),
				"successful-lrp-stop-auctions":  len(auctionResults.SuccessfulLRPStops),
				"failed-lrp-start-auctions":     len(auctionResults.FailedLRPStarts),
				"failed-lrp-stop-auctions":      len(auctionResults.FailedLRPStops),
			})
			numStartsFailed := len(auctionResults.FailedLRPStarts)
			numStopsFailed := len(auctionResults.FailedLRPStops)

			logger.Info("resubmitting-failures")
			auctionResults = ResubmitFailedAuctions(a.batch, auctionResults, a.maxRetries)
			logger.Info("resubmitted-failures", lager.Data{
				"successful-lrp-start-auctions":     len(auctionResults.SuccessfulLRPStarts),
				"successful-lrp-stop-auctions":      len(auctionResults.SuccessfulLRPStops),
				"will-not-retry-lrp-start-auctions": len(auctionResults.FailedLRPStarts),
				"will-not-retry-lrp-stop-auctions":  len(auctionResults.FailedLRPStops),
				"will-retry-lrp-start-auctions":     numStartsFailed - len(auctionResults.FailedLRPStarts),
				"will-retry-lrp-stop-auctions":      numStopsFailed - len(auctionResults.FailedLRPStops),
			})

			go a.delegate.DistributedBatch(auctionResults)
		case <-signals:
			return nil
		}
	}
}

func (a *auctionRunner) AddLRPStartAuction(lrpStart models.LRPStartAuction) {
	a.batch.AddLRPStartAuction(lrpStart)
}

func (a *auctionRunner) AddLRPStopAuction(lrpStop models.LRPStopAuction) {
	a.batch.AddLRPStopAuction(lrpStop)
}
