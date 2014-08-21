package inprocess

import (
	"math/rand"
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var LatencyMin time.Duration
var LatencyMax time.Duration

type InprocessClient struct {
	reps map[string]*auctionrep.AuctionRep
}

func New(reps map[string]*auctionrep.AuctionRep) *InprocessClient {
	return &InprocessClient{
		reps: reps,
	}
}

func (client *InprocessClient) beSlow(repGuid string) {
	sleepDuration := time.Duration(rand.Float64()*float64(LatencyMax-LatencyMin) + float64(LatencyMin))
	time.Sleep(sleepDuration)
}

func (client *InprocessClient) TotalResources(repGuid string) auctiontypes.Resources {
	return client.reps[repGuid].TotalResources()
}

func (client *InprocessClient) SimulatedInstances(repGuid string) []auctiontypes.SimulatedInstance {
	return client.reps[repGuid].SimulatedInstances()
}

func (client *InprocessClient) SetSimulatedInstances(repGuid string, instances []auctiontypes.SimulatedInstance) {
	client.reps[repGuid].SetSimulatedInstances(instances)
}

func (client *InprocessClient) Reset(repGuid string) {
	client.reps[repGuid].Reset()
}

func (client *InprocessClient) startAuctionBid(repGuid string, startAuctionInfo auctiontypes.StartAuctionInfo, c chan auctiontypes.StartAuctionBid) {
	result := auctiontypes.StartAuctionBid{
		Rep: repGuid,
	}
	defer func() {
		c <- result
	}()

	client.beSlow(repGuid)

	bid, err := client.reps[repGuid].BidForStartAuction(startAuctionInfo)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Bid = bid
	return
}

func (client *InprocessClient) stopScore(repGuid string, auctionInfo auctiontypes.StopAuctionInfo, c chan auctiontypes.StopAuctionBid) {
	result := auctiontypes.StopAuctionBid{
		Rep: repGuid,
	}
	defer func() {
		c <- result
	}()

	client.beSlow(repGuid)

	bid, instanceGuids, err := client.reps[repGuid].BidForStopAuction(auctionInfo)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.InstanceGuids = instanceGuids
	result.Bid = bid
}

func (client *InprocessClient) BidForStopAuction(representatives []string, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	c := make(chan auctiontypes.StopAuctionBid)
	for _, repGuid := range representatives {
		go client.stopScore(repGuid, stopAuctionInfo, c)
	}

	results := auctiontypes.StopAuctionBids{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) BidForStartAuction(representatives []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	c := make(chan auctiontypes.StartAuctionBid)
	for _, repGuid := range representatives {
		go client.startAuctionBid(repGuid, startAuctionInfo, c)
	}

	results := auctiontypes.StartAuctionBids{}
	for _ = range representatives {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) reserveAndRecastScore(repGuid string, startAuctionInfo auctiontypes.StartAuctionInfo, c chan auctiontypes.StartAuctionBid) {
	result := auctiontypes.StartAuctionBid{
		Rep: repGuid,
	}
	defer func() {
		c <- result
	}()

	client.beSlow(repGuid)

	bid, err := client.reps[repGuid].RebidThenTentativelyReserve(startAuctionInfo)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Bid = bid
	return
}

func (client *InprocessClient) RebidThenTentativelyReserve(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	c := make(chan auctiontypes.StartAuctionBid)
	for _, repGuid := range repGuids {
		go client.reserveAndRecastScore(repGuid, startAuctionInfo, c)
	}

	results := auctiontypes.StartAuctionBids{}
	for _ = range repGuids {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) ReleaseReservation(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) {
	c := make(chan bool)
	for _, repGuid := range repGuids {
		go func(repGuid string) {
			client.beSlow(repGuid)
			client.reps[repGuid].ReleaseReservation(startAuctionInfo)
			c <- true
		}(repGuid)
	}

	for _ = range repGuids {
		<-c
	}
}

func (client *InprocessClient) Run(repGuid string, startAuctionInfo models.LRPStartAuction) {
	client.beSlow(repGuid)

	client.reps[repGuid].Run(startAuctionInfo)
}

func (client *InprocessClient) Stop(repGuid string, stopInstance models.StopLRPInstance) {
	client.beSlow(repGuid)

	client.reps[repGuid].Stop(stopInstance)
}
