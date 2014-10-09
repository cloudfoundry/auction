package auctiondistributor

import (
	"fmt"
	"sync"

	"github.com/cheggaaa/pb"
	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
)

type inProcessAuctionDistributor struct {
	auctionRunner              auctiontypes.AuctionRunner
	maxConcurrentPerAuctioneer int
}

func NewInProcessAuctionDistributor(client auctiontypes.SimulationRepPoolClient, maxConcurrentPerAuctioneer int) AuctionDistributor {
	return &inProcessAuctionDistributor{
		auctionRunner:              auctionrunner.New(client),
		maxConcurrentPerAuctioneer: maxConcurrentPerAuctioneer,
	}
}

func (d *inProcessAuctionDistributor) HoldStartAuctions(numAuctioneers int, startAuctions []models.LRPStartAuction, repAddresses []auctiontypes.RepAddress, rules auctiontypes.StartAuctionRules) []auctiontypes.StartAuctionResult {
	startAuctionRequests := buildStartAuctionRequests(startAuctions, repAddresses, rules)
	numWorkers := d.maxConcurrentPerAuctioneer * numAuctioneers
	workPool := workpool.NewWorkPool(numWorkers)

	bar := pb.StartNew(len(startAuctions))

	wg := &sync.WaitGroup{}
	wg.Add(len(startAuctionRequests))
	results := []auctiontypes.StartAuctionResult{}
	lock := &sync.Mutex{}
	for _, startAuctionRequest := range startAuctionRequests {
		startAuctionRequest := startAuctionRequest
		workPool.Submit(func() {
			defer wg.Done()
			result, err := d.auctionRunner.RunLRPStartAuction(startAuctionRequest)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			lock.Lock()
			results = append(results, result)
			bar.Increment()
			lock.Unlock()
		})
	}

	wg.Wait()
	bar.Finish()
	workPool.Stop()
	return results
}

func (d *inProcessAuctionDistributor) HoldStopAuctions(numAuctioneers int, stopAuctions []models.LRPStopAuction, repAddresses []auctiontypes.RepAddress) []auctiontypes.StopAuctionResult {
	stopAuctionRequests := buildStopAuctionRequests(stopAuctions, repAddresses)
	numWorkers := d.maxConcurrentPerAuctioneer * numAuctioneers
	workPool := workpool.NewWorkPool(numWorkers)

	wg := &sync.WaitGroup{}
	wg.Add(len(stopAuctionRequests))
	results := []auctiontypes.StopAuctionResult{}
	lock := &sync.Mutex{}
	for _, stopAuctionRequest := range stopAuctionRequests {
		stopAuctionRequest := stopAuctionRequest
		workPool.Submit(func() {
			defer wg.Done()
			result, err := d.auctionRunner.RunLRPStopAuction(stopAuctionRequest)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			lock.Lock()
			results = append(results, result)
			lock.Unlock()
		})
	}
	wg.Wait()
	workPool.Stop()
	return results
}
