package auctiondistributor

import (
	"fmt"
	"time"

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
	maxConcurrent     int
}

func NewInProcessAuctionDistributor(client auctiontypes.SimulationRepPoolClient, maxConcurrent int) *AuctionDistributor {
	auctionRunner := auctionrunner.New(client)
	return &AuctionDistributor{
		client:        client,
		maxConcurrent: maxConcurrent,
		startCommunicator: func(auctionRequest auctiontypes.StartAuctionRequest) (auctiontypes.StartAuctionResult, error) {
			return auctionRunner.RunLRPStartAuction(auctionRequest)
		},
		stopCommunicator: func(auctionRequest auctiontypes.StopAuctionRequest) (auctiontypes.StopAuctionResult, error) {
			return auctionRunner.RunLRPStopAuction(auctionRequest)
		},
	}
}

func NewRemoteAuctionDistributor(hosts []string, client auctiontypes.SimulationRepPoolClient, maxConcurrent int) *AuctionDistributor {
	return &AuctionDistributor{
		client:            client,
		maxConcurrent:     maxConcurrent,
		startCommunicator: newHttpRemoteAuctions(hosts).RemoteStartAuction,
		stopCommunicator:  newHttpRemoteAuctions(hosts).RemoteStopAuction,
	}
}

func (ad *AuctionDistributor) HoldAuctionsFor(instances []models.LRPStartAuction, representatives []string, rules auctiontypes.AuctionRules) *visualization.Report {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))

	t := time.Now()
	semaphore := make(chan bool, ad.maxConcurrent)
	c := make(chan auctiontypes.StartAuctionResult)
	for _, inst := range instances {
		go func(inst models.LRPStartAuction) {
			semaphore <- true
			result, _ := ad.startCommunicator(auctiontypes.StartAuctionRequest{
				LRPStartAuction: inst,
				RepGuids:        representatives,
				Rules:           rules,
			})
			result.Duration = time.Since(t)
			c <- result
			<-semaphore
		}(inst)
	}

	results := []auctiontypes.StartAuctionResult{}
	for _ = range instances {
		results = append(results, <-c)
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
