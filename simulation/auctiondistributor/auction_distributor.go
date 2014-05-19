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

type AuctionCommunicator func(auctiontypes.AuctionRequest) (auctiontypes.AuctionResult, error)

type AuctionDistributor struct {
	client        auctiontypes.TestRepPoolClient
	communicator  AuctionCommunicator
	maxConcurrent int
}

func NewInProcessAuctionDistributor(client auctiontypes.TestRepPoolClient, maxConcurrent int) *AuctionDistributor {
	auctionRunner := auctionrunner.New(client)
	return &AuctionDistributor{
		client:        client,
		maxConcurrent: maxConcurrent,
		communicator: func(auctionRequest auctiontypes.AuctionRequest) (auctiontypes.AuctionResult, error) {
			return auctionRunner.RunLRPStartAuction(auctionRequest)
		},
	}
}

func NewRemoteAuctionDistributor(hosts []string, client auctiontypes.TestRepPoolClient, maxConcurrent int) *AuctionDistributor {
	return &AuctionDistributor{
		client:        client,
		maxConcurrent: maxConcurrent,
		communicator:  newHttpRemoteAuctions(hosts).RemoteAuction,
	}
}

func (ad *AuctionDistributor) HoldAuctionsFor(instances []models.LRPStartAuction, representatives []string, rules auctiontypes.AuctionRules) *visualization.Report {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))

	t := time.Now()
	semaphore := make(chan bool, ad.maxConcurrent)
	c := make(chan auctiontypes.AuctionResult)
	for _, inst := range instances {
		go func(inst models.LRPStartAuction) {
			semaphore <- true
			result, _ := ad.communicator(auctiontypes.AuctionRequest{
				LRPStartAuction: inst,
				RepGuids:        representatives,
				Rules:           rules,
			})
			result.Duration = time.Since(t)
			c <- result
			<-semaphore
		}(inst)
	}

	results := []auctiontypes.AuctionResult{}
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
