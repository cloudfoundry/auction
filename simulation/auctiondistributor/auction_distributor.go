package auctiondistributor

import (
	"fmt"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/onsi/auction/auctioneer"
	"github.com/onsi/auction/simulation/visualization"
	"github.com/onsi/auction/types"
)

type AuctionCommunicator func(types.AuctionRequest) types.AuctionResult

type AuctionDistributor struct {
	client        types.TestRepPoolClient
	communicator  AuctionCommunicator
	maxConcurrent int
}

func NewInProcessAuctionDistributor(client types.TestRepPoolClient, maxConcurrent int) *AuctionDistributor {
	return &AuctionDistributor{
		client:        client,
		maxConcurrent: maxConcurrent,
		communicator: func(auctionRequest types.AuctionRequest) types.AuctionResult {
			return auctioneer.Auction(client, auctionRequest)
		},
	}
}

func NewRemoteAuctionDistributor(hosts []string, client types.TestRepPoolClient, maxConcurrent int) *AuctionDistributor {
	return &AuctionDistributor{
		client:        client,
		maxConcurrent: maxConcurrent,
		communicator:  newHttpRemoteAuctions(hosts).RemoteAuction,
	}
}

func (ad *AuctionDistributor) HoldAuctionsFor(instances []types.Instance, representatives []string, rules types.AuctionRules) *visualization.Report {
	fmt.Printf("\nStarting Auctions\n\n")
	bar := pb.StartNew(len(instances))

	t := time.Now()
	semaphore := make(chan bool, ad.maxConcurrent)
	c := make(chan types.AuctionResult)
	for _, inst := range instances {
		go func(inst types.Instance) {
			semaphore <- true
			result := ad.communicator(types.AuctionRequest{
				Instance: inst,
				RepGuids: representatives,
				Rules:    rules,
			})
			result.Duration = time.Since(t)
			c <- result
			<-semaphore
		}(inst)
	}

	results := []types.AuctionResult{}
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
