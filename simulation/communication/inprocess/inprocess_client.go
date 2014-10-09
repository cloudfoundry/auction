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

func (client *InprocessClient) TotalResources(repAddress auctiontypes.RepAddress) auctiontypes.Resources {
	return client.reps[repAddress.RepGuid].TotalResources()
}

func (client *InprocessClient) SimulatedInstances(repAddress auctiontypes.RepAddress) []auctiontypes.SimulatedInstance {
	return client.reps[repAddress.RepGuid].SimulatedInstances()
}

func (client *InprocessClient) SetSimulatedInstances(repAddress auctiontypes.RepAddress, instances []auctiontypes.SimulatedInstance) {
	client.reps[repAddress.RepGuid].SetSimulatedInstances(instances)
}

func (client *InprocessClient) Reset(repAddress auctiontypes.RepAddress) {
	client.reps[repAddress.RepGuid].Reset()
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

func (client *InprocessClient) BidForStopAuction(repAddresses []auctiontypes.RepAddress, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	c := make(chan auctiontypes.StopAuctionBid)
	for _, repAddress := range repAddresses {
		go client.stopScore(repAddress.RepGuid, stopAuctionInfo, c)
	}

	results := auctiontypes.StopAuctionBids{}
	for _ = range repAddresses {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) BidForStartAuction(repAddresses []auctiontypes.RepAddress, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	c := make(chan auctiontypes.StartAuctionBid)
	for _, repAddress := range repAddresses {
		go client.startAuctionBid(repAddress.RepGuid, startAuctionInfo, c)
	}

	results := auctiontypes.StartAuctionBids{}
	for _ = range repAddresses {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) reserveAndRecastScore(repGuid string, startAuction models.LRPStartAuction, c chan auctiontypes.StartAuctionBid) {
	result := auctiontypes.StartAuctionBid{
		Rep: repGuid,
	}
	defer func() {
		c <- result
	}()

	client.beSlow(repGuid)

	bid, err := client.reps[repGuid].RebidThenTentativelyReserve(startAuction)
	if err != nil {
		result.Error = err.Error()
		return
	}

	result.Bid = bid
	return
}

func (client *InprocessClient) RebidThenTentativelyReserve(repAddresses []auctiontypes.RepAddress, startAuction models.LRPStartAuction) auctiontypes.StartAuctionBids {
	c := make(chan auctiontypes.StartAuctionBid)
	for _, repAddress := range repAddresses {
		go client.reserveAndRecastScore(repAddress.RepGuid, startAuction, c)
	}

	results := auctiontypes.StartAuctionBids{}
	for _ = range repAddresses {
		results = append(results, <-c)
	}

	return results
}

func (client *InprocessClient) ReleaseReservation(repAddresses []auctiontypes.RepAddress, startAuction models.LRPStartAuction) {
	c := make(chan bool)
	for _, repAddress := range repAddresses {
		go func(repGuid string) {
			client.beSlow(repGuid)
			client.reps[repGuid].ReleaseReservation(startAuction)
			c <- true
		}(repAddress.RepGuid)
	}

	for _ = range repAddresses {
		<-c
	}
}

func (client *InprocessClient) Run(repAddress auctiontypes.RepAddress, startAuctionInfo models.LRPStartAuction) {
	client.beSlow(repAddress.RepGuid)

	client.reps[repAddress.RepGuid].Run(startAuctionInfo)
}

func (client *InprocessClient) Stop(repAddress auctiontypes.RepAddress, stopInstance models.StopLRPInstance) {
	client.beSlow(repAddress.RepGuid)

	client.reps[repAddress.RepGuid].Stop(stopInstance)
}
