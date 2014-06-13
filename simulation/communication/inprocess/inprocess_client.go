package inprocess

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/util"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

var LatencyMin time.Duration
var LatencyMax time.Duration
var Timeout time.Duration

type InprocessClient struct {
	reps map[string]*auctionrep.AuctionRep
}

func New(reps map[string]*auctionrep.AuctionRep) *InprocessClient {
	return &InprocessClient{
		reps: reps,
	}
}

func randomSleep(min time.Duration, max time.Duration, timeout time.Duration) bool {
	sleepDuration := time.Duration(util.R.Float64()*float64(max-min) + float64(min))
	if sleepDuration <= timeout {
		time.Sleep(sleepDuration)
		return true
	} else {
		time.Sleep(timeout)
		return false
	}
}

func (client *InprocessClient) beSlowAndPossiblyTimeout(repGuid string) bool {
	sleepDuration := time.Duration(util.R.Float64()*float64(LatencyMax-LatencyMin) + float64(LatencyMin))

	if sleepDuration <= Timeout {
		time.Sleep(sleepDuration)
		return false
	} else {
		time.Sleep(Timeout)
		return true
	}
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

	if client.beSlowAndPossiblyTimeout(repGuid) {
		result.Error = "timeout"
		return
	}

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

	if client.beSlowAndPossiblyTimeout(repGuid) {
		result.Error = "timeout"
		return
	}

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

	if client.beSlowAndPossiblyTimeout(repGuid) {
		result.Error = "timedout"
		return
	}

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
			client.beSlowAndPossiblyTimeout(repGuid)
			client.reps[repGuid].ReleaseReservation(startAuctionInfo)
			c <- true
		}(repGuid)
	}

	for _ = range repGuids {
		<-c
	}
}

func (client *InprocessClient) Run(repGuid string, startAuctionInfo models.LRPStartAuction) {
	client.beSlowAndPossiblyTimeout(repGuid)

	client.reps[repGuid].Run(startAuctionInfo)
}

func (client *InprocessClient) Stop(repGuid string, stopInstance models.StopLRPInstance) {
	client.beSlowAndPossiblyTimeout(repGuid)

	client.reps[repGuid].Stop(stopInstance)
}
