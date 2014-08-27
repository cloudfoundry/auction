package auctiondistributor

import (
	"fmt"
	"runtime"
	"time"

	"github.com/onsi/ginkgo"

	"github.com/cheggaaa/pb"
	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/simulation/visualization"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type StartAuctionCommunicator func(auctiontypes.StartAuctionRequest) (auctiontypes.StartAuctionResult, error)
type StopAuctionCommunicator func(auctiontypes.StopAuctionRequest) (auctiontypes.StopAuctionResult, error)

type AuctionDistributor struct {
	client            auctiontypes.SimulationRepPoolClient
	startCommunicator StartAuctionCommunicator
	stopCommunicator  StopAuctionCommunicator
}

func NewInProcessAuctionDistributor(client auctiontypes.SimulationRepPoolClient) *AuctionDistributor {
	auctionRunner := auctionrunner.New(client)
	return &AuctionDistributor{
		client: client,
		startCommunicator: func(auctionRequest auctiontypes.StartAuctionRequest) (auctiontypes.StartAuctionResult, error) {
			return auctionRunner.RunLRPStartAuction(auctionRequest)
		},
		stopCommunicator: func(auctionRequest auctiontypes.StopAuctionRequest) (auctiontypes.StopAuctionResult, error) {
			return auctionRunner.RunLRPStopAuction(auctionRequest)
		},
	}
}

func NewRemoteAuctionDistributor(hosts []string, client auctiontypes.SimulationRepPoolClient) *AuctionDistributor {
	return &AuctionDistributor{
		client:            client,
		startCommunicator: newHttpRemoteAuctions(hosts).RemoteStartAuction,
		stopCommunicator:  newHttpRemoteAuctions(hosts).RemoteStopAuction,
	}
}

func (ad *AuctionDistributor) HoldAuctionsFor(instances []models.LRPStartAuction, representatives []string, rules auctiontypes.StartAuctionRules, maxConcurrent int) *visualization.Report {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))

	t := time.Now()
	workChan := make(chan models.LRPStartAuction)
	resultsChan := make(chan auctiontypes.StartAuctionResult, len(instances))

	for i := 0; i < maxConcurrent; i++ {
		go func(i int) {
			n := 0
			for inst := range workChan {
				n++
				startTime := time.Now()
				result, _ := ad.startCommunicator(auctiontypes.StartAuctionRequest{
					LRPStartAuction: inst,
					RepGuids:        representatives,
					Rules:           rules,
				})
				result.Duration = time.Since(t)
				resultsChan <- result
				fmt.Fprintln(ginkgo.GinkgoWriter, "Finished", inst.InstanceGuid, "on", i, "processed", n, "took:", time.Since(startTime), "communications:", result.NumCommunications, "numrounds:", result.NumRounds, "goroutines:", runtime.NumGoroutine(), "length:", len(resultsChan), "have-run:", n)
			}
			fmt.Fprintln(ginkgo.GinkgoWriter, i, "is gone - processed:", n)
		}(i)
	}

	for _, instance := range instances {
		workChan <- instance
	}

	close(workChan)

	results := []auctiontypes.StartAuctionResult{}
	for _ = range instances {
		results = append(results, <-resultsChan)
		bar.Increment()
	}

	bar.Finish()

	duration := time.Since(t)
	report := &visualization.Report{
		RepGuids:        representatives,
		AuctionResults:  results,
		InstancesByRep:  visualization.FetchAndSortInstances(ad.client, representatives),
		AuctionDuration: duration,
	}

	return report
}

func (ad *AuctionDistributor) HoldStopAuctions(stopAuctions []models.LRPStopAuction, representatives []string) []auctiontypes.StopAuctionResult {
	t := time.Now()

	c := make(chan auctiontypes.StopAuctionResult)
	for _, stopAuction := range stopAuctions {
		go func(stopAuction models.LRPStopAuction) {
			result, _ := ad.stopCommunicator(auctiontypes.StopAuctionRequest{
				LRPStopAuction: stopAuction,
				RepGuids:       representatives,
			})
			result.Duration = time.Since(t)
			c <- result
		}(stopAuction)
	}

	results := []auctiontypes.StopAuctionResult{}
	for _ = range stopAuctions {
		results = append(results, <-c)
	}

	return results
}
